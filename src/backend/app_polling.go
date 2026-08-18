package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robinson/gos7"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// StartPolling starts polling loop according to current AppSettings
func (a *App) StartPolling(settings AppSettings) {
	a.StopPolling()

	a.mu.Lock()
	a.settings = settings
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	pollCtx, cancel := context.WithCancel(baseCtx)
	a.cancelPoll = cancel
	a.isPolling = true
	done := make(chan struct{})
	a.pollDone = done
	a.mu.Unlock()

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
				for {
					select {
					case <-pollCtx.Done():
						return
					case <-ticker.C:
						for _, ch := range linkTags {
							a.emitPollUpdate(ch.Id, "", nil, "Bad", fmt.Sprintf("PLC link '%s' not configured", lName))
						}
					}
				}
			}

			type parsedTag struct {
				Tag  TagSettings
				Spec *S7AddressSpec
				Err  error
			}

			var pTags []parsedTag
			for _, ch := range linkTags {
				spec, err := ParseS7Address(ch.Address, ch.DataType, ch.StringLength)
				pTags = append(pTags, parsedTag{Tag: ch, Spec: spec, Err: err})
			}

			type networkResult struct {
				Items      []gos7.S7DataItem
				DurationMs int
				Err        error
			}
			dataChan := make(chan networkResult, 100)

			// Goroutine 1: The Network Worker (Paced by user sample interval, using AGReadMulti)
			go func() {
				pollTicker := time.NewTicker(time.Duration(interval) * time.Millisecond)
				defer pollTicker.Stop()

				for {
					select {
					case <-pollCtx.Done():
						return
					case <-pollTicker.C:
						if !a.CheckStatus(lName) {
							err := a.Connect(lName, link.IpAddress, link.Rack, link.Slot)
							if err != nil {
								select {
								case <-pollCtx.Done():
									return
								case dataChan <- networkResult{Err: err}:
								}
								continue
							}
						}

						a.mu.RLock()
						conn := a.plcs[lName]
						a.mu.RUnlock()

						if conn == nil {
							select {
							case <-pollCtx.Done():
								return
							case dataChan <- networkResult{Err: fmt.Errorf("Not connected")}:
							}
							continue
						}

						var dataItems []gos7.S7DataItem
						var validIndices []int

						for i, pt := range pTags {
							if pt.Err == nil {
								dataItems = append(dataItems, gos7.S7DataItem{
									Area:     pt.Spec.Area,
									WordLen:  0x02, // Byte length
									DBNumber: pt.Spec.DbNumber,
									Start:    pt.Spec.StartByte,
									Amount:   pt.Spec.ByteLength,
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
							case dataChan <- networkResult{Err: fmt.Errorf("Not connected")}:
							}
							continue
						}

						readStart := time.Now()
						err := conn.Client.AGReadMulti(dataItems, len(dataItems))
						readDurationMs := int(time.Since(readStart).Milliseconds())
						if readDurationMs == 0 {
							readDurationMs = int(time.Since(readStart).Microseconds()) / 1000
							if readDurationMs < 1 {
								readDurationMs = 1
							}
						}
						conn.mu.Unlock()

						if err != nil {
							a.invalidateConnection(lName, conn)
							select {
							case <-pollCtx.Done():
								return
							case dataChan <- networkResult{Err: err, DurationMs: readDurationMs}:
							}
						} else {
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
							case dataChan <- networkResult{Items: fullItems, DurationMs: readDurationMs, Err: nil}:
							}
						}
					}
				}
			}()

			// Goroutine 2: The Consumer & Wails Emitter (Throttles updates for Wails IPC bridge if necessary)
			uiInterval := time.Duration(interval) * time.Millisecond
			if uiInterval < 33*time.Millisecond {
				uiInterval = 33 * time.Millisecond
			}
			ticker := time.NewTicker(uiInterval)
			defer ticker.Stop()

			var latestData []gos7.S7DataItem
			var latestErr error

			for {
				select {
				case <-pollCtx.Done():
					return
				case res := <-dataChan:
					latestData = res.Items
					latestErr = res.Err

					if res.DurationMs > 0 {
						a.emitPollTiming(lName, res.DurationMs)
					}

				case <-ticker.C:
					if latestErr != nil {
						for _, pt := range pTags {
							a.emitPollUpdate(pt.Tag.Id, "", nil, "Bad", latestErr.Error())
						}
						latestErr = nil
					} else if latestData != nil {
						for i, pt := range pTags {
							if pt.Err != nil {
								a.emitPollUpdate(pt.Tag.Id, "", nil, "Bad", pt.Err.Error())
								continue
							}
							item := latestData[i]
							if item.Error != "" {
								a.emitPollUpdate(pt.Tag.Id, "", nil, "Bad", item.Error)
							} else {
								valStr, numVal := DecodeS7Value(item.Data, pt.Tag.DataType, pt.Spec.BitNumber)
								a.emitPollUpdate(pt.Tag.Id, valStr, numVal, "Good", "")
							}
						}
						latestData = nil
					}
				}
			}
		}(targetLink, tags, linkName)
	}

	go func() {
		wg.Wait()
		close(done)
	}()
}

func (a *App) emitPollUpdate(tagId uuid.UUID, valStr string, numVal *float64, quality string, errStr string) {
	update := PollUpdate{
		Timestamp:    time.Now().Format(time.RFC3339Nano),
		Value:        valStr,
		NumericValue: numVal,
		Quality:      quality,
		Error:        errStr,
	}

	update.TagId = tagId

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "poll_update", update)
	}
}

func (a *App) emitPollTiming(plcLink string, actualIntervalMs int) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "poll_timing", PollTiming{
		PlcLink:          plcLink,
		ActualIntervalMs: actualIntervalMs,
	})
}

func (a *App) StopPolling() {
	a.mu.Lock()
	cancel := a.cancelPoll
	done := a.pollDone
	a.cancelPoll = nil
	a.pollDone = nil
	a.isPolling = false
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (a *App) invalidateConnection(linkName string, conn *PlcConnection) {
	a.mu.Lock()
	defer a.mu.Unlock()
	current, ok := a.plcs[linkName]
	if !ok || current != conn {
		return
	}
	conn.mu.Lock()
	if conn.IsConnected {
		_ = conn.Handler.Close()
		conn.IsConnected = false
	}
	conn.mu.Unlock()
	delete(a.plcs, linkName)
}


