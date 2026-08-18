package backend

import (
	"context"
	"testing"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("expected NewApp() to return non-nil")
	}
	if app.plcs == nil {
		t.Error("expected plcs map to be initialized")
	}
	s := app.GetSettings()
	if s.PollIntervalMs != 100 {
		t.Errorf("expected initial PollIntervalMs 100, got %d", s.PollIntervalMs)
	}
	if len(s.PlcLinks) != 4 {
		t.Errorf("expected 4 PlcLinks, got %d", len(s.PlcLinks))
	}
}

func TestApp_GetVersion(t *testing.T) {
	app := NewApp()
	v := app.GetVersion()
	if v == "" {
		t.Error("expected non-empty version string")
	}
}

func TestApp_GetSystemLanguage(t *testing.T) {
	app := NewApp()
	lang := app.GetSystemLanguage()
	if lang == "" {
		t.Error("expected non-empty language string")
	}
}

func TestApp_Settings(t *testing.T) {
	app := NewApp()
	original := app.GetSettings()

	modified := original
	modified.PollIntervalMs = 500
	modified.TimeWindowSeconds = 180
	modified.Interpolation = InterpolationDifferential

	app.SaveSettings(modified)

	current := app.GetSettings()
	if current.PollIntervalMs != 500 {
		t.Errorf("expected PollIntervalMs 500, got %d", current.PollIntervalMs)
	}
	if current.TimeWindowSeconds != 180 {
		t.Errorf("expected TimeWindowSeconds 180, got %d", current.TimeWindowSeconds)
	}
	if current.Interpolation != InterpolationDifferential {
		t.Errorf("expected Interpolation Differential, got %s", current.Interpolation)
	}
}

func TestFindLink(t *testing.T) {
	links := []PlcLinkSettings{
		{Name: "PLC1", IpAddress: "192.168.0.1"},
		{Name: "PLC2", IpAddress: "192.168.0.2"},
	}

	found := findLink(links, "PLC2")
	if found == nil || found.IpAddress != "192.168.0.2" {
		t.Errorf("expected to find PLC2, got %+v", found)
	}

	notFound := findLink(links, "NonExistent")
	if notFound != nil {
		t.Errorf("expected nil for non-existent link, got %+v", notFound)
	}
}

func TestApp_CheckStatus_NonExistent(t *testing.T) {
	app := NewApp()
	if app.CheckStatus("UnknownPLC") {
		t.Error("expected CheckStatus for unknown PLC to return false")
	}
}

func TestApp_StartupShutdown(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.Startup(ctx)
	if app.ctx != ctx {
		t.Errorf("app.ctx not set properly on Startup")
	}

	// Should not panic on shutdown
	app.Shutdown(ctx)
}

func TestApp_Disconnect_NonExistent(t *testing.T) {
	app := NewApp()
	err := app.Disconnect("NonExistent")
	if err != nil {
		t.Errorf("disconnecting non-existent link should return nil, got %v", err)
	}
}
