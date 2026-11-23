package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robinson/gos7"
)

const (
	plcLinkStateUnknown = iota
	plcLinkStateConnected
	plcLinkStateDisconnected

	maxPendingBatchUpdates = 1024
)

type parsedTag struct {
	Tag  TagSettings
	Spec *S7AddressSpec
	Err  error
}

type networkResult struct {
	Items      []gos7.S7DataItem
	DurationMs int
	ReadAt     time.Time
	Err        error
}

// StartPolling starts polling loop according to current AppSettings
func (a *App) StartPolling(settings AppSettings) {
	wasPolling := a.stopPollingInternal(false)
	settings = normalizeSettings(settings)

	a.mu.Lock()
	oldLinks := append([]PlcLinkSettings(nil), a.settings.PlcLinks...)
	a.settings = settings
	pollCtx, cancel := context.WithCancel(context.Background())
	a.cancelPoll = cancel
	a.isPolling = true
	done := make(chan struct{})
	a.pollDone = done
	a.mu.Unlock()

	for _, old := range oldLinks {
		current := findLink(settings.PlcLinks, old.Name)
		if current == nil || old.IpAddress != current.IpAddress || old.Rack != current.Rack || old.Slot != current.Slot {
			_ = a.Disconnect(old.Name)
		}
	}
	a.refreshTrayStatus()
	if !wasPolling {
		a.emitAppEvent("app:sampling-state", SamplingStateEvent{IsSampling: true})
	}

	interval := settings.PollIntervalMs
	if interval < 10 {
		interval = 10
	}
	if interval > 60000 {
		interval = 60000
	}

	tagsByPlc := make(map[string][]TagSettings)
	for _, ch := range settings.Tags {
		if !ch.Enabled {
			continue
		}
		tagsByPlc[ch.PlcLink] = append(tagsByPlc[ch.PlcLink], ch)
	}

	var wg sync.WaitGroup

	for linkName, tags := range tagsByPlc {
		var targetLink *PlcLinkSettings
		for idx := range settings.PlcLinks {
			if settings.PlcLinks[idx].Name == linkName {
				targetLink = &settings.PlcLinks[idx]
				break
			}
		}

		wg.Add(1)
		go func(link *PlcLinkSettings, linkTags []TagSettings, lName string) {
			defer wg.Done()

			if link == nil {
				ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
				defer ticker.Stop()
				errMsg := fmt.Sprintf("PLC link '%s' not configured", lName)
				for {
					select {
					case <-pollCtx.Done():
						return
					case <-ticker.C:
						nowStr := time.Now().Format(time.RFC3339Nano)
						updates := make([]PollUpdate, 0, len(linkTags))
						for _, ch := range linkTags {
							updates = append(updates, newPollUpdate(ch.Id, nowStr, "", nil, "Bad", errMsg))
						}
						a.publishTelemetry(TelemetryFrame{Updates: updates})
					}
				}
			}

			var pTags []parsedTag
			validCount := 0
			for _, ch := range linkTags {
				spec, err := ParseS7Address(ch.Address, ch.DataType)
				if err == nil {
					if validCount >= MaxTagsPerPlcLink {
						err = fmt.Errorf("exceeded max %d tags per PLC link (configure another PLC connection for additional tags)", MaxTagsPerPlcLink)
					} else {
						validCount++
					}
				}
				pTags = append(pTags, parsedTag{Tag: ch, Spec: spec, Err: err})
			}

			dataChan := make(chan networkResult, 100)
			var linkWg sync.WaitGroup
			linkWg.Add(1)

			// Goroutine 1: The Network Worker (Paced by user sample interval, using AGReadMulti)
			go func() {
				defer linkWg.Done()
				pollTicker := time.NewTicker(time.Duration(interval) * time.Millisecond)
				defer pollTicker.Stop()

				linkState := plcLinkStateUnknown
				if a.CheckStatus(lName) {
					linkState = plcLinkStateConnected
				}

				var (
					backoff        = 1 * time.Second
					maxBackoff     = 30 * time.Second
					nextRetryTime  time.Time
					lastConnectErr error
				)

				for {
					select {
					case <-pollCtx.Done():
						return
					case <-pollTicker.C:
						if !a.CheckStatus(lName) {
							now := time.Now()
							if !nextRetryTime.IsZero() && now.Before(nextRetryTime) {
								// Currently in backoff delay; do not flood reconnect attempts
								select {
								case <-pollCtx.Done():
									return
								case dataChan <- networkResult{Err: lastConnectErr, ReadAt: now}:
								}
								continue
							}

							err := a.Connect(lName, link.IpAddress, link.Rack, link.Slot)
							if err != nil {
								lastConnectErr = err
								nextRetryTime = now.Add(backoff)
								backoff *= 2
								if backoff > maxBackoff {
									backoff = maxBackoff
								}
								if pollCtx.Err() == nil {
									a.notifyPlcStateTransition(lName, link.IpAddress, linkState, plcLinkStateDisconnected, err)
									linkState = plcLinkStateDisconnected
								}
								select {
								case <-pollCtx.Done():
									return
								case dataChan <- networkResult{Err: err, ReadAt: now}:
								}
								continue
							}

							// Connection succeeded; reset backoff
							backoff = 1 * time.Second
							nextRetryTime = time.Time{}
							lastConnectErr = nil
							if pollCtx.Err() == nil && linkState != plcLinkStateConnected {
								a.notifyPlcStateTransition(lName, link.IpAddress, linkState, plcLinkStateConnected, nil)
								linkState = plcLinkStateConnected
							}
						}

						a.mu.RLock()
						conn := a.plcs[lName]
						a.mu.RUnlock()

						if conn == nil {
							select {
							case <-pollCtx.Done():
								return
							case dataChan <- networkResult{Err: fmt.Errorf("Not connected"), ReadAt: time.Now()}:
							}
							continue
						}

						var dataItems []gos7.S7DataItem
						var validIndices []int

						for i, pt := range pTags {
							if pt.Err == nil {
								wordLen := pt.Spec.WordLen
								if wordLen == 0 {
									wordLen = 0x02 // Byte length
								}
								amount := pt.Spec.ByteLength
								if pt.Spec.Area == s7AreaTM || pt.Spec.Area == s7AreaCT {
									amount = 1 // 1 timer/counter element
								}
								dataItems = append(dataItems, gos7.S7DataItem{
									Area:     pt.Spec.Area,
									WordLen:  wordLen,
									DBNumber: pt.Spec.DbNumber,
									Start:    pt.Spec.StartByte,
									Amount:   amount,
									Data:     make([]byte, pt.Spec.ByteLength),
								})
								validIndices = append(validIndices, i)
							}
						}

						if len(dataItems) == 0 {
							continue
						}

						conn.mu.Lock()
						if !conn.IsConnected {
							conn.mu.Unlock()
							select {
							case <-pollCtx.Done():
								return
							case dataChan <- networkResult{Err: fmt.Errorf("Not connected"), ReadAt: time.Now()}:
							}
							continue
						}

						readStart := time.Now()
						err := conn.Client.AGReadMulti(dataItems, len(dataItems))
						readTime := time.Now()
						readDurationMs := int(readTime.Sub(readStart).Milliseconds())
						if readDurationMs == 0 {
							readDurationMs = int(readTime.Sub(readStart).Microseconds()) / 1000
							if readDurationMs < 1 {
								readDurationMs = 1
							}
						}
						conn.mu.Unlock()

						if err != nil {
							a.invalidateConnection(lName, conn)
							if pollCtx.Err() == nil {
								a.notifyPlcStateTransition(lName, link.IpAddress, linkState, plcLinkStateDisconnected, err)
								linkState = plcLinkStateDisconnected
							}
							lastConnectErr = err
							nextRetryTime = time.Now().Add(backoff)
							backoff *= 2
							if backoff > maxBackoff {
								backoff = maxBackoff
							}
							select {
							case <-pollCtx.Done():
								return
							case dataChan <- networkResult{Err: err, DurationMs: readDurationMs, ReadAt: readTime}:
							}
						} else {
							backoff = 1 * time.Second
							nextRetryTime = time.Time{}
							lastConnectErr = nil
							if pollCtx.Err() == nil && linkState != plcLinkStateConnected {
								a.notifyPlcStateTransition(lName, link.IpAddress, linkState, plcLinkStateConnected, nil)
								linkState = plcLinkStateConnected
							}

							fullItems := make([]gos7.S7DataItem, len(pTags))
							for idx, pt := range pTags {
								if pt.Err != nil {
									fullItems[idx] = gos7.S7DataItem{Error: pt.Err.Error()}
								}
							}
							for i, dItem := range dataItems {
								fullItems[validIndices[i]] = dItem
							}

							select {
							case <-pollCtx.Done():
								return
							case dataChan <- networkResult{Items: fullItems, DurationMs: readDurationMs, ReadAt: readTime, Err: nil}:
							}
						}
					}
				}
			}()

			// Goroutine 2: The Consumer & Stream Publisher (Records every sample immediately and batches UI/WS frames)
			defer linkWg.Wait()
			a.runLinkConsumer(pollCtx, lName, pTags, interval, dataChan)
		}(targetLink, tags, linkName)
	}

	go func() {
		wg.Wait()
		close(done)
	}()
}

func (a *App) runLinkConsumer(ctx context.Context, linkName string, pTags []parsedTag, intervalMs int, dataChan <-chan networkResult) {
	uiInterval := time.Duration(intervalMs) * time.Millisecond
	if uiInterval < 33*time.Millisecond {
		uiInterval = 33 * time.Millisecond
	}
	ticker := time.NewTicker(uiInterval)
	defer ticker.Stop()

	var (
		pendingUpdates   []PollUpdate
		latestDurationMs int
		pendingErr       error
		pendingErrTime   time.Time
	)

	flush := func() {
		updates := pendingUpdates
		pendingUpdates = nil

		if pendingErr != nil {
			errStr := pendingErr.Error()
			errTime := pendingErrTime
			if errTime.IsZero() {
				errTime = time.Now()
			}
			nowStr := errTime.Format(time.RFC3339Nano)
			for _, pt := range pTags {
				updates = append(updates, newPollUpdate(pt.Tag.Id, nowStr, "", nil, "Bad", errStr))
			}
			pendingErr = nil
			pendingErrTime = time.Time{}
		}

		var timings []PollTiming
		if latestDurationMs > 0 {
			timings = []PollTiming{{
				PlcLink:          linkName,
				ActualIntervalMs: latestDurationMs,
			}}
			latestDurationMs = 0
		}

		if len(updates) > 0 || len(timings) > 0 {
			a.publishTelemetry(TelemetryFrame{
				Updates: updates,
				Timings: timings,
			})
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-dataChan:
			if !ok {
				flush()
				return
			}
			if res.DurationMs > 0 {
				latestDurationMs = res.DurationMs
			}
			sampleTime := res.ReadAt
			if sampleTime.IsZero() {
				sampleTime = time.Now()
			}

			if res.Err != nil {
				pendingErr = res.Err
				pendingErrTime = sampleTime
			} else if res.Items != nil {
				pendingErr = nil
				pendingErrTime = time.Time{}

				nowMs := sampleTime.UnixMilli()
				nowStr := sampleTime.Format(time.RFC3339Nano)
				for i, pt := range pTags {
					if pt.Err != nil {
						pendingUpdates = append(pendingUpdates, newPollUpdate(pt.Tag.Id, nowStr, "", nil, "Bad", pt.Err.Error()))
						continue
					}
					item := res.Items[i]
					if item.Error != "" {
						pendingUpdates = append(pendingUpdates, newPollUpdate(pt.Tag.Id, nowStr, "", nil, "Bad", item.Error))
						continue
					}
					var valStr string
					var numVal *float64
					if pt.Spec.Area == s7AreaTM {
						valStr, numVal = DecodeS7Timer(item.Data)
					} else if pt.Spec.Area == s7AreaCT {
						valStr, numVal = DecodeS7Counter(item.Data)
					} else {
						valStr, numVal = DecodeS7Value(item.Data, pt.Tag.DataType, pt.Spec.BitNumber)
					}
					if numVal != nil {
						a.recordPolledSample(pt.Tag, nowMs, *numVal)
					}
					pendingUpdates = append(pendingUpdates, newPollUpdate(pt.Tag.Id, nowStr, valStr, numVal, "Good", ""))
				}
				if len(pendingUpdates) >= maxPendingBatchUpdates {
					flush()
				}
			}
		case <-ticker.C:
			flush()
		}
	}
}

func newPollUpdate(tagId uuid.UUID, timestamp, valStr string, numVal *float64, quality, errStr string) PollUpdate {
	return PollUpdate{
		TagId:        tagId,
		Timestamp:    timestamp,
		Value:        valStr,
		NumericValue: numVal,
		Quality:      quality,
		Error:        errStr,
	}
}

func (a *App) emitPollUpdate(tagId uuid.UUID, valStr string, numVal *float64, quality string, errStr string) {
	update := newPollUpdate(tagId, time.Now().Format(time.RFC3339Nano), valStr, numVal, quality, errStr)
	a.publishTelemetry(TelemetryFrame{Updates: []PollUpdate{update}})
}

func (a *App) emitPollTiming(plcLink string, actualIntervalMs int) {
	a.publishTelemetry(TelemetryFrame{
		Timings: []PollTiming{{
			PlcLink:          plcLink,
			ActualIntervalMs: actualIntervalMs,
		}},
	})
}

func (a *App) notifyPlcStateTransition(linkName, ip string, prevState, nextState int, err error) {
	if prevState == nextState {
		return
	}
	host := a.getHost()
	if host == nil {
		return
	}

	switch {
	case prevState == plcLinkStateUnknown && nextState == plcLinkStateDisconnected:
		host.Notify("PLC Connection Failed", fmt.Sprintf("%s (%s): %v", linkName, ip, err))
	case prevState == plcLinkStateConnected && nextState == plcLinkStateDisconnected:
		host.Notify("PLC Disconnected", fmt.Sprintf("%s (%s): %v", linkName, ip, err))
	case prevState == plcLinkStateDisconnected && nextState == plcLinkStateConnected:
		host.Notify("PLC Reconnected", fmt.Sprintf("%s (%s) connection restored", linkName, ip))
	}
}

func (a *App) StopPolling() {
	a.stopPollingInternal(true)
}

func (a *App) stopPollingInternal(emitTransition bool) bool {
	a.mu.Lock()
	wasPolling := a.isPolling
	cancel := a.cancelPoll
	done := a.pollDone
	a.cancelPoll = nil
	a.pollDone = nil
	a.isPolling = false
	a.mu.Unlock()
	a.refreshTrayStatus()
	if emitTransition && wasPolling {
		a.emitAppEvent("app:sampling-state", SamplingStateEvent{IsSampling: false})
	}
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
	return wasPolling
}

func (a *App) invalidateConnection(linkName string, conn *PlcConnection) {
	a.mu.Lock()
	current, ok := a.plcs[linkName]
	if !ok || current != conn {
		a.mu.Unlock()
		return
	}
	conn.mu.Lock()
	if conn.IsConnected {
		_ = conn.Handler.Close()
		conn.IsConnected = false
	}
	conn.mu.Unlock()
	delete(a.plcs, linkName)
	a.mu.Unlock()

	a.refreshTrayStatus()
}
