package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type trayStatusUpdate struct {
	isSampling    bool
	connectedPlcs int
	totalPlcs     int
}

type toastNotification struct {
	title string
	body  string
}

type webServerTrayUpdate struct {
	running bool
	addr    string
}

type mockDesktopHost struct {
	mu              sync.Mutex
	emittedEvents   []string
	emittedPayloads []any
	trayUpdates     []trayStatusUpdate
	webUpdates      []webServerTrayUpdate
	notifications   []toastNotification
	quitCalled      bool
	showCalled      bool
	hideCalled      bool
	bounds          struct {
		w, h, x, y int
		ok         bool
	}
}

func (m *mockDesktopHost) EmitEvent(name string, data ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emittedEvents = append(m.emittedEvents, name)
	if len(data) > 0 {
		m.emittedPayloads = append(m.emittedPayloads, data[0])
	} else {
		m.emittedPayloads = append(m.emittedPayloads, nil)
	}
}
func (m *mockDesktopHost) Quit() {
	m.mu.Lock()
	m.quitCalled = true
	m.mu.Unlock()
}
func (m *mockDesktopHost) ShowWindow() {
	m.mu.Lock()
	m.showCalled = true
	m.mu.Unlock()
}
func (m *mockDesktopHost) HideWindow() {
	m.mu.Lock()
	m.hideCalled = true
	m.mu.Unlock()
}
func (m *mockDesktopHost) GetWindowBounds() (int, int, int, int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bounds.w, m.bounds.h, m.bounds.x, m.bounds.y, m.bounds.ok
}
func (m *mockDesktopHost) SetWindowBounds(int, int, int, int, bool) {}
func (m *mockDesktopHost) OpenFileDialog(string, string, string, string) (string, error) {
	return "", nil
}
func (m *mockDesktopHost) SaveFileDialog(string, string, string, string, string) (string, error) {
	return "", nil
}
func (m *mockDesktopHost) UpdateTrayStatus(isSampling bool, connectedPlcs, totalPlcs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trayUpdates = append(m.trayUpdates, trayStatusUpdate{
		isSampling:    isSampling,
		connectedPlcs: connectedPlcs,
		totalPlcs:     totalPlcs,
	})
}
func (m *mockDesktopHost) UpdateTrayWebServer(running bool, addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webUpdates = append(m.webUpdates, webServerTrayUpdate{
		running: running,
		addr:    addr,
	})
}
func (m *mockDesktopHost) Notify(title, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, toastNotification{
		title: title,
		body:  body,
	})
}

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
	host := &mockDesktopHost{}
	AttachHost(app, host)
	if app.getHost() != host {
		t.Errorf("app.host not set properly on AttachHost")
	}

	// Should not panic on ServiceShutdown (and remain idempotent)
	if err := app.ServiceShutdown(); err != nil {
		t.Errorf("unexpected ServiceShutdown error: %v", err)
	}
	if err := app.ServiceShutdown(); err != nil {
		t.Errorf("unexpected second ServiceShutdown error: %v", err)
	}
}

func TestApp_Disconnect_NonExistent(t *testing.T) {
	app := NewApp()
	err := app.Disconnect("NonExistent")
	if err != nil {
		t.Errorf("disconnecting non-existent link should return nil, got %v", err)
	}
}

func TestApp_HistoryMethods(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	// Record samples for tag-1 and tag-2
	app.RecordSample("tag-1", 1000, 10.5)
	app.RecordSample("tag-1", 2000, 11.0)
	app.RecordSample("tag-1", 3000, 12.5)
	app.RecordSample("tag-2", 2000, 99.9)
	app.Flush()

	// Query range [1500, 2500]
	res, err := app.GetHistoryRange(context.Background(), []string{"tag-1", "tag-2", "tag-3"}, 1500, 2500, 0)
	if err != nil {
		t.Fatalf("unexpected GetHistoryRange error: %v", err)
	}
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
	resAfter, err := app.GetHistoryRange(context.Background(), []string{"tag-1"}, 0, 5000, 0)
	if err != nil {
		t.Fatalf("unexpected GetHistoryRange error after ClearHistory: %v", err)
	}
	if len(resAfter["tag-1"]) != 0 {
		t.Errorf("expected tag-1 to have 0 points after ClearHistory, got %+v", resAfter["tag-1"])
	}
}

func TestApp_GetHistoryRangeFlushesPendingSamples(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	app.RecordSample("tag-1", 1000, 42.5)

	res, err := app.GetHistoryRange(context.Background(), []string{"tag-1"}, 0, 2000, 0)
	if err != nil {
		t.Fatalf("unexpected GetHistoryRange error: %v", err)
	}
	if len(res["tag-1"]) != 1 || res["tag-1"][0].Timestamp != 1000 || res["tag-1"][0].Value != 42.5 {
		t.Fatalf("expected GetHistoryRange to flush and return pending sample immediately, got %+v", res["tag-1"])
	}
}

func TestApp_GetHistoryRangeCancelledContext(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	app.RecordSample("tag-1", 1000, 42.5)
	app.Flush()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := app.GetHistoryRange(ctx, []string{"tag-1"}, 0, 2000, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled for cancelled GetHistoryRange call, got %v", err)
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

	if err := app.SaveCurrentSettings(); err != nil {
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

func TestApp_BeforeClose(t *testing.T) {
	app := NewApp()
	host := &mockDesktopHost{}
	AttachHost(app, host)

	// Settings not changed, tray disabled -> HandleWindowClosing returns false (allow exit)
	if HandleWindowClosing(app) {
		t.Error("expected HandleWindowClosing to return false when settings have not changed")
	}

	// Modify settings, tray disabled -> HandleWindowClosing returns true and emits app:request-exit
	s := app.GetSettings()
	s.PollIntervalMs = 999
	app.SaveSettings(s)

	if !HandleWindowClosing(app) {
		t.Error("expected HandleWindowClosing to return true when settings changed")
	}
	if len(host.emittedEvents) == 0 || host.emittedEvents[len(host.emittedEvents)-1] != "app:request-exit" {
		t.Errorf("expected app:request-exit event, got %v", host.emittedEvents)
	}

	// Tray enabled -> HandleWindowClosing hides window and prevents close
	SetTrayEnabled(app, true)
	if !HandleWindowClosing(app) {
		t.Error("expected HandleWindowClosing to return true when tray is enabled")
	}
	if !host.hideCalled {
		t.Error("expected HideWindow to be called when tray is enabled")
	}

	// Tray exit while dirty -> shows window and emits app:request-exit
	host.emittedEvents = nil
	RequestExitFromTray(app)
	if !host.showCalled {
		t.Error("expected ShowWindow to be called on RequestExitFromTray when dirty")
	}
	if len(host.emittedEvents) != 1 || host.emittedEvents[0] != "app:request-exit" {
		t.Errorf("expected app:request-exit on RequestExitFromTray, got %v", host.emittedEvents)
	}

	// Force exit -> QuitApp sets forceExit and calls host.Quit(); HandleWindowClosing allows exit
	app.QuitApp()
	if !host.quitCalled {
		t.Error("expected host.Quit to be called by QuitApp")
	}
	if HandleWindowClosing(app) {
		t.Error("expected HandleWindowClosing to return false when forceExit is true")
	}

	// Tray exit when clean -> calls host.Quit() directly
	cleanApp := NewApp()
	cleanHost := &mockDesktopHost{}
	AttachHost(cleanApp, cleanHost)
	RequestExitFromTray(cleanApp)
	if !cleanHost.quitCalled {
		t.Error("expected host.Quit to be called when RequestExitFromTray runs with clean settings")
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

func TestApp_ToggleSamplingFromTrayAndTrayStatus(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	host := &mockDesktopHost{}
	AttachHost(app, host)

	host.mu.Lock()
	if len(host.trayUpdates) == 0 || host.trayUpdates[0].isSampling {
		host.mu.Unlock()
		t.Fatalf("expected AttachHost to initialize idle tray status, got %+v", host.trayUpdates)
	}
	host.mu.Unlock()

	ToggleSamplingFromTray(app)
	if !app.IsPolling() {
		t.Fatal("expected IsPolling to be true after first tray toggle")
	}

	ToggleSamplingFromTray(app)
	if app.IsPolling() {
		t.Fatal("expected IsPolling to be false after second tray toggle")
	}

	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.emittedEvents) < 2 || host.emittedEvents[0] != "app:sampling-state" || host.emittedEvents[1] != "app:sampling-state" {
		t.Fatalf("expected two app:sampling-state events, got %v", host.emittedEvents)
	}
	firstPayload, ok1 := host.emittedPayloads[0].(SamplingStateEvent)
	secondPayload, ok2 := host.emittedPayloads[1].(SamplingStateEvent)
	if !ok1 || !firstPayload.IsSampling || !ok2 || secondPayload.IsSampling {
		t.Fatalf("unexpected app:sampling-state payloads: %+v", host.emittedPayloads)
	}
	if len(host.trayUpdates) < 3 {
		t.Fatalf("expected tray status updates across Attach/Start/Stop, got %+v", host.trayUpdates)
	}
	last := host.trayUpdates[len(host.trayUpdates)-1]
	if last.isSampling {
		t.Errorf("expected final tray update to be idle, got %+v", last)
	}
}

func TestApp_LoadSettingsFromPathAndFindArg(t *testing.T) {
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	host := &mockDesktopHost{}
	AttachHost(app, host)

	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "dropped.json")
	custom := CreateDefaultSettings()
	custom.PollIntervalMs = 250
	custom.TimeWindowSeconds = 120
	custom.Interpolation = InterpolationStep
	raw, err := json.MarshalIndent(custom, "", "  ")
	if err != nil {
		t.Fatalf("marshal custom settings: %v", err)
	}
	if err := os.WriteFile(validPath, raw, 0644); err != nil {
		t.Fatalf("write valid settings: %v", err)
	}

	// Verify FindSettingsJSONArg resolves relative path against workingDir and skips flags/dirs/non-json
	if got := FindSettingsJSONArg([]string{"--flag", "notes.txt", tmpDir, "dropped.json"}, tmpDir); got != validPath {
		t.Fatalf("expected FindSettingsJSONArg to return %s, got %s", validPath, got)
	}

	// Start polling so we also verify LoadSettingsFromPath restarts active polling
	app.StartPolling(CreateDefaultSettings())
	if err := LoadSettingsFromPath(app, validPath); err != nil {
		t.Fatalf("LoadSettingsFromPath valid failed: %v", err)
	}
	if !app.IsPolling() {
		t.Error("expected polling to remain active after LoadSettingsFromPath")
	}
	app.StopPolling()

	got := app.GetSettings()
	if got.PollIntervalMs != 250 || got.TimeWindowSeconds != 120 || got.Interpolation != InterpolationStep {
		t.Errorf("unexpected loaded settings: %+v", got)
	}
	if app.HasSettingsChanged() {
		t.Error("expected HasSettingsChanged == false immediately after LoadSettingsFromPath")
	}

	// Non-object JSON should fail and emit settings:load-error
	invalidPath := filepath.Join(tmpDir, "array.json")
	if err := os.WriteFile(invalidPath, []byte(`[1, 2, 3]`), 0644); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	if err := LoadSettingsFromPath(app, invalidPath); err == nil {
		t.Fatal("expected error loading non-object JSON")
	}

	host.mu.Lock()
	defer host.mu.Unlock()
	var sawLoaded, sawLoadError bool
	for _, ev := range host.emittedEvents {
		switch ev {
		case "settings:loaded":
			sawLoaded = true
		case "settings:load-error":
			sawLoadError = true
		}
	}
	if !sawLoaded || !sawLoadError {
		t.Fatalf("expected settings:loaded and settings:load-error events, got %v", host.emittedEvents)
	}
}

func TestApp_ImportSettingsJSON_WriteCSV_StatusSnapshot_AndTransitions(t *testing.T) {
	isolateConfigDir(t)
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	host := &mockDesktopHost{}
	AttachHost(app, host)

	// Calling StopPolling when already idle must not emit duplicate transition event
	app.StopPolling()
	host.mu.Lock()
	for _, ev := range host.emittedEvents {
		if ev == "app:sampling-state" {
			host.mu.Unlock()
			t.Fatalf("did not expect app:sampling-state when stopping idle app, got %v", host.emittedEvents)
		}
	}
	host.mu.Unlock()

	if _, err := app.ImportSettingsJSON(`[1,2,3]`); err == nil {
		t.Fatal("expected error for non-object JSON in ImportSettingsJSON")
	}

	tagID := uuid.New()
	custom := CreateDefaultSettings()
	custom.PollIntervalMs = 300
	custom.Tags = []TagSettings{
		{
			Id:       tagID,
			Name:     "Flow",
			PlcLink:  "PLC1",
			Address:  "DB2.DBD4",
			DataType: DataTypeReal,
			YAxis:    "Y-Axis 1",
			Color:    "#00ff00",
			Enabled:  true,
		},
	}
	raw, err := json.Marshal(custom)
	if err != nil {
		t.Fatalf("marshal settings failed: %v", err)
	}

	applied, err := app.ImportSettingsJSON(string(raw))
	if err != nil {
		t.Fatalf("ImportSettingsJSON failed: %v", err)
	}
	if applied.PollIntervalMs != 300 || len(applied.Tags) != 1 || applied.Tags[0].Name != "Flow" {
		t.Fatalf("unexpected ImportSettingsJSON result: %+v", applied)
	}

	ts := time.Date(2026, 6, 1, 8, 30, 0, 0, time.UTC).UnixMilli()
	app.RecordSample(tagID.String(), ts, 18.75)

	var csvBuf bytes.Buffer
	if err := WriteCSV(app, &csvBuf); err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}
	csvStr := csvBuf.String()
	if !strings.Contains(csvStr, "Timestamp,Tag,Address,Value") || !strings.Contains(csvStr, "Flow,DB2.DBD4,18.75") {
		t.Fatalf("unexpected CSV output:\n%s", csvStr)
	}

	snap := StatusSnapshot(app)
	if snap.Version != AppVersion || snap.IsPolling || snap.TotalPlcs != 1 || snap.EnabledTags != 1 || snap.PollIntervalMs != 300 {
		t.Fatalf("unexpected StatusSnapshot: %+v", snap)
	}
}
