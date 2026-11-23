package backend

import "sync"

const defaultTelemetryBufferSize = 64

// AppEvent represents a named backend event broadcast to desktop and web clients.
type AppEvent struct {
	Name string `json:"name"`
	Data any    `json:"data,omitempty"`
}

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

func (a *App) subscribeEvents(buffer int) (<-chan AppEvent, func()) {
	if buffer <= 0 {
		buffer = defaultTelemetryBufferSize
	}

	ch := make(chan AppEvent, buffer)

	a.eventMu.Lock()
	if a.eventSubs == nil {
		a.eventSubs = make(map[uint64]chan AppEvent)
	}
	a.eventSeq++
	id := a.eventSeq
	a.eventSubs[id] = ch
	a.eventMu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			a.eventMu.Lock()
			sub, ok := a.eventSubs[id]
			if ok {
				delete(a.eventSubs, id)
			}
			a.eventMu.Unlock()
			if ok {
				close(sub)
			}
		})
	}

	return ch, unsubscribe
}

func (a *App) publishAppEvent(name string, data any) {
	ev := AppEvent{Name: name, Data: data}
	a.eventMu.RLock()
	for _, ch := range a.eventSubs {
		select {
		case ch <- ev:
		default:
		}
	}
	a.eventMu.RUnlock()
}

func (a *App) closeAllEventSubscribers() {
	a.eventMu.Lock()
	subs := a.eventSubs
	a.eventSubs = make(map[uint64]chan AppEvent)
	a.eventMu.Unlock()

	for _, ch := range subs {
		close(ch)
	}
}

func (a *App) emitAppEvent(name string, data any) {
	if host := a.getHost(); host != nil {
		if data == nil {
			host.EmitEvent(name)
		} else {
			host.EmitEvent(name, data)
		}
	}
	a.publishAppEvent(name, data)
}
