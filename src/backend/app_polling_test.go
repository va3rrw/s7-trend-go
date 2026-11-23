package backend

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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
