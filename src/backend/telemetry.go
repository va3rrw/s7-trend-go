package backend

import "sync"

const defaultTelemetryBufferSize = 64

// SubscribeTelemetry registers a subscriber channel that receives batched live
// telemetry frames during polling. Slow consumers drop frames rather than
// stalling PLC polling goroutines.
//
//wails:ignore
func (a *App) SubscribeTelemetry(buffer int) (<-chan TelemetryFrame, func()) {
	if buffer <= 0 {
		buffer = defaultTelemetryBufferSize
	}

	ch := make(chan TelemetryFrame, buffer)

	a.telemetryMu.Lock()
	if a.telemetrySubs == nil {
		a.telemetrySubs = make(map[uint64]chan TelemetryFrame)
	}
	a.telemetrySeq++
	id := a.telemetrySeq
	a.telemetrySubs[id] = ch
	a.telemetryMu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			a.telemetryMu.Lock()
			sub, ok := a.telemetrySubs[id]
			if ok {
				delete(a.telemetrySubs, id)
			}
			a.telemetryMu.Unlock()
			if ok {
				close(sub)
			}
		})
	}

	return ch, unsubscribe
}

func (a *App) publishTelemetry(frame TelemetryFrame) {
	if len(frame.Updates) == 0 && len(frame.Timings) == 0 {
		return
	}

	a.telemetryMu.RLock()
	for _, ch := range a.telemetrySubs {
		select {
		case ch <- frame:
		default:
		}
	}
	a.telemetryMu.RUnlock()
}

func (a *App) closeAllTelemetrySubscribers() {
	a.telemetryMu.Lock()
	subs := a.telemetrySubs
	a.telemetrySubs = make(map[uint64]chan TelemetryFrame)
	a.telemetryMu.Unlock()

	for _, ch := range subs {
		close(ch)
	}
}
