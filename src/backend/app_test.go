package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if len(s.PlcLinks) != 1 {
		t.Errorf("expected 1 PlcLink, got %d", len(s.PlcLinks))
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
	modified.Interpolation = InterpolationStep

	app.SaveSettings(modified)

	current := app.GetSettings()
	if current.PollIntervalMs != 500 {
		t.Errorf("expected PollIntervalMs 500, got %d", current.PollIntervalMs)
	}
	if current.TimeWindowSeconds != 180 {
		t.Errorf("expected TimeWindowSeconds 180, got %d", current.TimeWindowSeconds)
	}
	if current.Interpolation != InterpolationStep {
		t.Errorf("expected Interpolation Step, got %s", current.Interpolation)
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

func TestApp_HistoryMethods(t *testing.T) {
	app := NewApp()

	// Record samples for tag-1 and tag-2
	app.RecordSample("tag-1", 1000, 10.5)
	app.RecordSample("tag-1", 2000, 11.0)
	app.RecordSample("tag-1", 3000, 12.5)
	app.RecordSample("tag-2", 2000, 99.9)

	// Query range [1500, 2500]
	res := app.GetHistoryRange([]string{"tag-1", "tag-2", "tag-3"}, 1500, 2500)
	if len(res["tag-1"]) != 1 || res["tag-1"][0].Timestamp != 2000 {
		t.Errorf("expected tag-1 to have 1 point at 2000, got %+v", res["tag-1"])
	}
	if len(res["tag-2"]) != 1 || res["tag-2"][0].Value != 99.9 {
		t.Errorf("expected tag-2 to have 1 point at 99.9, got %+v", res["tag-2"])
	}
	if len(res["tag-3"]) != 0 {
		t.Errorf("expected tag-3 to have 0 points, got %+v", res["tag-3"])
	}

	// Clear history
	app.ClearHistory()
	resAfter := app.GetHistoryRange([]string{"tag-1"}, 0, 5000)
	if len(resAfter["tag-1"]) != 0 {
		t.Errorf("expected tag-1 to have 0 points after ClearHistory, got %+v", resAfter["tag-1"])
	}
}

func TestGetDefaultSettingsDir(t *testing.T) {
	dir := getDefaultSettingsDir()
	if dir == "" {
		t.Fatal("expected non-empty default settings directory")
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "/s7-trend-go") {
		t.Errorf("expected directory to end with /s7-trend-go, got %s", dir)
	}
}

func TestApp_SettingsChangedAndState(t *testing.T) {
	app := NewApp()
	if app.HasSettingsChanged() {
		t.Error("expected new app to have HasSettingsChanged == false")
	}

	// Change settings
	s := app.GetSettings()
	s.PollIntervalMs = 750
	app.SaveSettings(s)

	if !app.HasSettingsChanged() {
		t.Error("expected HasSettingsChanged == true after modifying settings")
	}

	// Create temp settings file
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "custom-settings.json")

	// Save settings directly to file and mark saved
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal settings: %v", err)
	}
	if err := os.WriteFile(settingsFile, data, 0644); err != nil {
		t.Fatalf("failed to write settings file: %v", err)
	}

	app.mu.Lock()
	app.lastSettingsPath = settingsFile
	app.savedSettingsJSON = app.serializeSettingsLocked()
	app.mu.Unlock()
	app.saveAppState(settingsFile)

	if app.HasSettingsChanged() {
		t.Error("expected HasSettingsChanged == false after marking settings saved")
	}
	if app.GetLastSettingsPath() != settingsFile {
		t.Errorf("expected lastSettingsPath %s, got %s", settingsFile, app.GetLastSettingsPath())
	}

	// SaveCurrentSettings should update the file directly
	s.PollIntervalMs = 1200
	app.SaveSettings(s)
	if !app.HasSettingsChanged() {
		t.Error("expected HasSettingsChanged == true after second modification")
	}

	if err := app.SaveCurrentSettings(context.Background()); err != nil {
		t.Fatalf("SaveCurrentSettings failed: %v", err)
	}

	if app.HasSettingsChanged() {
		t.Error("expected HasSettingsChanged == false after SaveCurrentSettings")
	}

	// Verify loaded file content on disk
	readBack, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read back settings file: %v", err)
	}
	var loadedSettings AppSettings
	if err := json.Unmarshal(readBack, &loadedSettings); err != nil {
		t.Fatalf("failed to unmarshal saved settings: %v", err)
	}
	if loadedSettings.PollIntervalMs != 1200 {
		t.Errorf("expected saved PollIntervalMs 1200, got %d", loadedSettings.PollIntervalMs)
	}
}

func TestApp_RecordSample_Concurrent(t *testing.T) {
	app := NewApp()
	done := make(chan struct{})

	// Concurrently record samples
	for i := 0; i < 8; i++ {
		go func(worker int) {
			for j := 0; j < 500; j++ {
				tagId := fmt.Sprintf("tag-%d", j%4)
				app.RecordSample(tagId, int64(j*10), float64(worker*100+j))
			}
			done <- struct{}{}
		}(i)
	}

	// Concurrently clear history
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Millisecond)
			app.ClearHistory()
		}
		done <- struct{}{}
	}()

	for i := 0; i < 9; i++ {
		<-done
	}
}




