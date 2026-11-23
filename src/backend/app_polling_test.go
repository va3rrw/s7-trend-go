package backend

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robinson/gos7"
)

func TestApp_StartStopPolling_EmptyTags(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()

	app.StartPolling(settings)
	if !app.isPolling {
		t.Error("expected app.isPolling to be true after StartPolling")
	}

	time.Sleep(50 * time.Millisecond)

	app.StopPolling()
	if app.isPolling {
		t.Error("expected app.isPolling to be false after StopPolling")
	}
}

func TestApp_StartStopPolling_Multiple(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()

	// Calling StopPolling when not polling should be a safe no-op
	app.StopPolling()

	app.StartPolling(settings)
	// Calling StartPolling again should safely stop previous and restart
	app.StartPolling(settings)

	time.Sleep(30 * time.Millisecond)
	app.StopPolling()
	app.StopPolling()
}

func TestApp_StartPolling_DisabledTags(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()
	settings.Tags = []TagSettings{
		{
			Id:       uuid.New(),
			Name:     "DisabledTag",
			PlcLink:  "PLC1",
			Address:  "DB1.DBD0",
			DataType: DataTypeReal,
			Enabled:  false,
		},
	}

	app.StartPolling(settings)
	time.Sleep(50 * time.Millisecond)
	app.StopPolling()
}

func TestApp_StartPolling_UnconfiguredLink(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()
	// Tag references PLC99 which is not in settings.PlcLinks
	settings.Tags = []TagSettings{
		{
			Id:       uuid.New(),
			Name:     "OrphanTag",
			PlcLink:  "PLC99",
			Address:  "DB1.DBD0",
			DataType: DataTypeReal,
			Enabled:  true,
		},
	}

	app.StartPolling(settings)
	time.Sleep(100 * time.Millisecond)
	app.StopPolling()
}

func TestApp_EmitMethods_NilHost(t *testing.T) {
	app := NewApp()
	val := 12.34
	// Neither of these should panic when app.host is nil
	app.emitPollUpdate(uuid.New(), "12.34", &val, "Good", "")
	app.emitPollTiming("PLC1", 50)
}

func TestApp_InvalidateConnection(t *testing.T) {
	app := NewApp()
	// Invalidate non-existent connection
	app.invalidateConnection("PLC1", nil)

	conn := &PlcConnection{
		IsConnected: false,
	}
	app.plcs["PLC1"] = conn
	app.invalidateConnection("PLC1", conn)

	if _, exists := app.plcs["PLC1"]; exists {
		t.Error("expected PLC1 to be removed after invalidateConnection")
	}
}

func TestApp_StartPolling_MaxTagsPerPlcLinkCap(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()

	// Configure 20 enabled tags on PLC1 (which exceeds MaxTagsPerPlcLink = 16)
	var tags []TagSettings
	for i := 0; i < 20; i++ {
		tags = append(tags, TagSettings{
			Id:       uuid.New(),
			Name:     "Tag",
			PlcLink:  "PLC1",
			Address:  "DB1.DBD0",
			DataType: DataTypeReal,
			Enabled:  true,
		})
	}
	settings.Tags = tags

	app.StartPolling(settings)
	time.Sleep(50 * time.Millisecond)
	app.StopPolling()
}

func TestApp_StartPolling_UpdateTagsWhilePolling(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()
	app.StartPolling(settings)

	if !app.isPolling {
		t.Fatal("expected isPolling to be true")
	}

	// Update tags while sampling is running
	newSettings := settings
	newSettings.Tags = append(newSettings.Tags, TagSettings{
		Id:       uuid.New(),
		Name:     "NewTagLive",
		PlcLink:  "PLC1",
		Address:  "DB1.DBD4",
		DataType: DataTypeReal,
		Enabled:  true,
	})

	app.StartPolling(newSettings)

	if !app.isPolling {
		t.Error("expected isPolling to remain true after StartPolling with updated tags")
	}
	if len(app.GetSettings().Tags) != len(newSettings.Tags) {
		t.Errorf("expected %d tags, got %d", len(newSettings.Tags), len(app.GetSettings().Tags))
	}

	time.Sleep(50 * time.Millisecond)
	app.StopPolling()
}

func TestApp_SaveSettings_WhilePolling(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()
	app.StartPolling(settings)

	if !app.isPolling {
		t.Fatal("expected isPolling to be true")
	}

	newSettings := settings
	newSettings.PollIntervalMs = 250
	newSettings.Tags = append(newSettings.Tags, TagSettings{
		Id:       uuid.New(),
		Name:     "TagFromSaveSettings",
		PlcLink:  "PLC1",
		Address:  "DB1.DBD8",
		DataType: DataTypeReal,
		Enabled:  true,
	})

	app.SaveSettings(newSettings)

	if !app.isPolling {
		t.Error("expected isPolling to remain true after SaveSettings while polling")
	}
	if app.GetSettings().PollIntervalMs != 250 {
		t.Errorf("expected PollIntervalMs 250, got %d", app.GetSettings().PollIntervalMs)
	}

	time.Sleep(50 * time.Millisecond)
	app.StopPolling()
}

func TestApp_StartPolling_ReconnectionBackoff(t *testing.T) {
	app := NewApp()
	settings := CreateDefaultSettings()
	settings.PollIntervalMs = 10
	settings.PlcLinks = []PlcLinkSettings{
		{Name: "PLC1", IpAddress: "127.0.0.1", Rack: 0, Slot: 1}, // Unreachable S7 port
	}
	settings.Tags = []TagSettings{
		{
			Id:       uuid.New(),
			Name:     "Tag1",
			PlcLink:  "PLC1",
			Address:  "DB1.DBD0",
			DataType: DataTypeReal,
			Enabled:  true,
		},
	}

	app.StartPolling(settings)
	time.Sleep(100 * time.Millisecond)
	app.StopPolling()
}

func TestApp_TelemetryStreamBatchingAndUnsubscribe(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	slowCh, slowUnsub := app.SubscribeTelemetry(1)
	defer slowUnsub()

	fastCh, fastUnsub := app.SubscribeTelemetry(8)
	defer fastUnsub()

	tagA := uuid.New()
	tagB := uuid.New()
	valA := 12.5
	valB := 25.0

	// Publishing two updates in one frame + one timing frame must deliver
	// batched updates to subscribers and never block when slowCh fills up.
	app.publishTelemetry(TelemetryFrame{
		Updates: []PollUpdate{
			newPollUpdate(tagA, "2026-01-01T00:00:00Z", "12.5", &valA, "Good", ""),
			newPollUpdate(tagB, "2026-01-01T00:00:00Z", "25", &valB, "Good", ""),
		},
		Timings: []PollTiming{{PlcLink: "PLC1", ActualIntervalMs: 18}},
	})
	app.emitPollTiming("PLC1", 22)
	app.emitPollUpdate(tagA, "15", &valA, "Good", "")

	// Fast subscriber received all 3 frames without drops.
	var received []TelemetryFrame
	for i := 0; i < 3; i++ {
		select {
		case frame := <-fastCh:
			received = append(received, frame)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for frame %d on fast subscriber", i)
		}
	}
	if len(received[0].Updates) != 2 || len(received[0].Timings) != 1 {
		t.Fatalf("expected batched first frame with 2 updates and 1 timing, got %+v", received[0])
	}
	if len(received[1].Timings) != 1 || received[1].Timings[0].ActualIntervalMs != 22 {
		t.Fatalf("expected second frame timing 22ms, got %+v", received[1])
	}
	if len(received[2].Updates) != 1 || received[2].Updates[0].Value != "15" {
		t.Fatalf("expected third frame update 15, got %+v", received[2])
	}

	// Slow subscriber buffer (cap=1) retained first frame and dropped overflow without blocking.
	<-slowCh
	select {
	case extra := <-slowCh:
		t.Fatalf("expected overflow frames to be dropped for full buffer, got %+v", extra)
	default:
	}

	// Unsubscribe must be idempotent and close the channel.
	fastUnsub()
	fastUnsub()
	if _, ok := <-fastCh; ok {
		t.Fatal("expected fastCh to be closed after unsubscribe")
	}

	// Verify real unconfigured-link polling batches all tags for a tick in a single frame.
	batchCh, batchUnsub := app.SubscribeTelemetry(16)
	defer batchUnsub()
	settings := CreateDefaultSettings()
	settings.PollIntervalMs = 20
	settings.Tags = []TagSettings{
		{Id: tagA, Name: "A", PlcLink: "MISSING", Address: "DB1.DBD0", DataType: DataTypeReal, Enabled: true},
		{Id: tagB, Name: "B", PlcLink: "MISSING", Address: "DB1.DBD4", DataType: DataTypeReal, Enabled: true},
	}
	app.StartPolling(settings)
	defer app.StopPolling()

	select {
	case frame := <-batchCh:
		if len(frame.Updates) != 2 {
			t.Fatalf("expected single tick frame batching 2 tag updates, got %d (%+v)", len(frame.Updates), frame)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for batched polling telemetry frame")
	}
}

func TestApp_PlcStateTransitionNotifications(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	host := &mockDesktopHost{}
	AttachHost(app, host)

	connErr := errors.New("connection refused")
	// Startup connecting normally (unknown -> connected) must stay silent.
	app.notifyPlcStateTransition("PLC1", "127.0.0.1", plcLinkStateUnknown, plcLinkStateConnected, nil)
	// First connection failure (unknown -> disconnected) notifies once.
	app.notifyPlcStateTransition("PLC1", "127.0.0.1", plcLinkStateUnknown, plcLinkStateDisconnected, connErr)
	// Repeated backoff failures (disconnected -> disconnected) must NOT spam.
	app.notifyPlcStateTransition("PLC1", "127.0.0.1", plcLinkStateDisconnected, plcLinkStateDisconnected, connErr)
	app.notifyPlcStateTransition("PLC1", "127.0.0.1", plcLinkStateDisconnected, plcLinkStateDisconnected, connErr)
	// Recovery (disconnected -> connected) notifies once.
	app.notifyPlcStateTransition("PLC1", "127.0.0.1", plcLinkStateDisconnected, plcLinkStateConnected, nil)
	// Live disconnect (connected -> disconnected) notifies once.
	app.notifyPlcStateTransition("PLC1", "127.0.0.1", plcLinkStateConnected, plcLinkStateDisconnected, connErr)

	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.notifications) != 3 {
		t.Fatalf("expected exactly 3 transition notifications, got %d: %+v", len(host.notifications), host.notifications)
	}
	if host.notifications[0].title != "PLC Connection Failed" ||
		host.notifications[1].title != "PLC Reconnected" ||
		host.notifications[2].title != "PLC Disconnected" {
		t.Fatalf("unexpected notification titles: %+v", host.notifications)
	}
}

func TestApp_SubscribeTelemetryConcurrentRace(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	var wg sync.WaitGroup
	tagID := uuid.New()
	val := 1.0

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsub := app.SubscribeTelemetry(8)
			defer unsub()
			for j := 0; j < 20; j++ {
				select {
				case <-ch:
				default:
				}
				time.Sleep(time.Millisecond)
			}
		}()
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				app.emitPollUpdate(tagID, "1", &val, "Good", "")
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
}

func TestApp_PollingConsumerBatchesHighFrequencySamples(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	fastReal := TagSettings{
		Id:                 uuid.New(),
		Name:               "FastSpike",
		PlcLink:            "PLC1",
		Address:            "DB1.DBD0",
		DataType:           DataTypeReal,
		Enabled:            true,
		SamplingIntervalMs: 0,
	}
	fastBool := TagSettings{
		Id:                 uuid.New(),
		Name:               "FastPulse",
		PlcLink:            "PLC1",
		Address:            "DB1.DBX4.0",
		DataType:           DataTypeBool,
		Enabled:            true,
		SamplingIntervalMs: 0,
	}
	slowReal := TagSettings{
		Id:                 uuid.New(),
		Name:               "SlowLevel",
		PlcLink:            "PLC1",
		Address:            "DB1.DBD8",
		DataType:           DataTypeReal,
		Enabled:            true,
		SamplingIntervalMs: 500,
	}

	settings := CreateDefaultSettings()
	settings.PollIntervalMs = 10
	settings.Tags = []TagSettings{fastReal, fastBool, slowReal}
	app.SaveSettings(settings)

	var pTags []parsedTag
	for _, tag := range settings.Tags {
		spec, err := ParseS7Address(tag.Address, tag.DataType)
		if err != nil {
			t.Fatalf("ParseS7Address(%s) failed: %v", tag.Address, err)
		}
		pTags = append(pTags, parsedTag{Tag: tag, Spec: spec})
	}

	encodeReal := func(v float32) []byte {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, math.Float32bits(v))
		return b
	}
	encodeBool := func(v bool) []byte {
		if v {
			return []byte{1}
		}
		return []byte{0}
	}

	t0 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	dataChan := make(chan networkResult, 8)
	samples := []struct {
		at       time.Time
		durMs    int
		fast     float32
		pulse    bool
		slow     float32
	}{
		{t0, 4, 10, false, 100},
		// +9ms simulates 1ms ticker jitter on a 10ms poll cadence
		{t0.Add(9 * time.Millisecond), 3, 99.5, true, 200},
		{t0.Add(20 * time.Millisecond), 5, 12, false, 300},
	}
	for _, s := range samples {
		dataChan <- networkResult{
			DurationMs: s.durMs,
			ReadAt:     s.at,
			Items: []gos7.S7DataItem{
				{Data: encodeReal(s.fast)},
				{Data: encodeBool(s.pulse)},
				{Data: encodeReal(s.slow)},
			},
		}
	}
	close(dataChan)

	ch, unsub := app.SubscribeTelemetry(8)
	defer unsub()

	app.runLinkConsumer(context.Background(), "PLC1", pTags, 10, dataChan)

	var frame TelemetryFrame
	select {
	case frame = <-ch:
	default:
		t.Fatal("expected single flushed TelemetryFrame when dataChan closes")
	}
	select {
	case extra := <-ch:
		t.Fatalf("expected all 3 reads batched into 1 frame, got extra frame: %+v", extra)
	default:
	}

	if len(frame.Updates) != 9 {
		t.Fatalf("expected 9 updates (3 polls x 3 tags), got %d", len(frame.Updates))
	}
	if len(frame.Timings) != 1 || frame.Timings[0].PlcLink != "PLC1" || frame.Timings[0].ActualIntervalMs != 5 {
		t.Fatalf("expected single consolidated timing ActualIntervalMs=5, got %+v", frame.Timings)
	}

	var fastValues, boolValues []float64
	for _, u := range frame.Updates {
		if u.Quality != "Good" || u.NumericValue == nil {
			t.Fatalf("unexpected non-good update: %+v", u)
		}
		switch u.TagId {
		case fastReal.Id:
			fastValues = append(fastValues, *u.NumericValue)
		case fastBool.Id:
			boolValues = append(boolValues, *u.NumericValue)
		}
	}
	if len(fastValues) != 3 || fastValues[0] != 10 || fastValues[1] != 99.5 || fastValues[2] != 12 {
		t.Fatalf("expected FastSpike sequence [10 99.5 12], got %v", fastValues)
	}
	if len(boolValues) != 3 || boolValues[0] != 0 || boolValues[1] != 1 || boolValues[2] != 0 {
		t.Fatalf("expected FastPulse sequence [0 1 0], got %v", boolValues)
	}

	hist, err := app.GetHistoryRange(
		context.Background(),
		[]string{fastReal.Id.String(), fastBool.Id.String(), slowReal.Id.String()},
		t0.UnixMilli()-100,
		t0.UnixMilli()+1000,
		0,
	)
	if err != nil {
		t.Fatalf("GetHistoryRange failed: %v", err)
	}
	if pts := hist[fastReal.Id.String()]; len(pts) != 3 || pts[1].Value != 99.5 {
		t.Fatalf("expected 3 history points preserving 99.5 spike at +9ms, got %+v", pts)
	}
	if pts := hist[fastBool.Id.String()]; len(pts) != 3 || pts[1].Value != 1 || pts[2].Value != 0 {
		t.Fatalf("expected 3 history points preserving bool pulse [0 1 0], got %+v", pts)
	}
	if pts := hist[slowReal.Id.String()]; len(pts) != 1 || pts[0].Value != 100 {
		t.Fatalf("expected 500ms slow tag to record 1 sample in 20ms window, got %+v", pts)
	}

	// Verify consecutive errors within one UI batch coalesce into 1 Bad update per tag
	errChan := make(chan networkResult, 4)
	for i := 0; i < 3; i++ {
		errChan <- networkResult{
			Err:    errors.New("timeout"),
			ReadAt: t0.Add(time.Duration(30+i*10) * time.Millisecond),
		}
	}
	close(errChan)

	app.runLinkConsumer(context.Background(), "PLC1", pTags, 10, errChan)

	var errFrame TelemetryFrame
	select {
	case errFrame = <-ch:
	default:
		t.Fatal("expected single flushed error TelemetryFrame")
	}
	if len(errFrame.Updates) != 3 {
		t.Fatalf("expected 3 coalesced Bad updates (1 per tag), got %d", len(errFrame.Updates))
	}
	for _, u := range errFrame.Updates {
		if u.Quality != "Bad" || u.Error != "timeout" {
			t.Fatalf("expected Bad/timeout update, got %+v", u)
		}
	}
}
