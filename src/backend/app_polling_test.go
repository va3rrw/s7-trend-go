package backend

import (
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

func TestApp_EmitMethods_NilContext(t *testing.T) {
	app := NewApp()
	val := 12.34
	// Neither of these should panic when app.ctx is nil
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

