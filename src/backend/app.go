package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jeandeaual/go-locale"
	"github.com/robinson/gos7"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetSystemLanguage() string {
	loc, err := locale.GetLanguage()
	if err != nil {
		return "en"
	}
	return loc
}

// GetVersion returns the app version (bumped by build.sh on each build).
func (a *App) GetVersion() string {
	return AppVersion
}

type PlcConnection struct {
	Client      gos7.Client
	Handler     *gos7.TCPClientHandler
	IsConnected bool
	mu          sync.Mutex
}

type App struct {
	ctx               context.Context
	plcs              map[string]*PlcConnection
	settings          AppSettings
	savedSettingsJSON string
	lastSettingsPath  string
	forceExit         bool
	mu                sync.RWMutex
	cancelPoll        context.CancelFunc
	isPolling         bool
	pollDone          chan struct{}

	history       *HistoryStore
	historyErr    error
	sampleMu      sync.Mutex
	lastSampleAt  map[string]int64
	lastBoolValue map[string]float64
	windowWidth   int
	windowHeight  int
	windowX       int
	windowY       int
	windowPosSet  bool

	trayMu      sync.Mutex
	trayEnabled bool
	trayIcon    []byte
	tray        *trayController
}

func NewApp() *App {
	return newApp(defaultHistoryDatabasePath())
}

// NewAppWithHistoryPath is useful for isolated deployments and tests that need
// a database outside the user's normal application data directory.
func NewAppWithHistoryPath(historyPath string) *App {
	return newApp(historyPath)
}

func newApp(historyPath string) *App {
	history, historyErr := OpenHistoryStore(historyPath)
	app := &App{
		plcs:          make(map[string]*PlcConnection),
		settings:      CreateDefaultSettings(),
		history:       history,
		historyErr:    historyErr,
		lastSampleAt:  make(map[string]int64),
		lastBoolValue: make(map[string]float64),
	}
	app.savedSettingsJSON = app.serializeSettingsLocked()
	app.loadAppState()
	return app
}

func (a *App) serializeSettingsLocked() string {
	data, err := json.Marshal(a.settings)
	if err != nil {
		return ""
	}
	return string(data)
}

// HasSettingsChanged returns true if current in-memory settings differ from last saved baseline
func (a *App) HasSettingsChanged() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.serializeSettingsLocked() != a.savedSettingsJSON
}

// GetLastSettingsPath returns the path to the currently active / last-used settings file
func (a *App) GetLastSettingsPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSettingsPath
}

// QuitApp forces application exit without prompting
func (a *App) QuitApp() {
	a.mu.Lock()
	a.forceExit = true
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

func (a *App) requestExitFromTray() {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return
	}

	if a.HasSettingsChanged() {
		runtime.WindowShow(ctx)
		runtime.WindowUnminimise(ctx)
		runtime.EventsEmit(ctx, "app:request-exit")
		return
	}
	a.QuitApp()
}

// EnableTray enables close-to-tray behavior and uses iconBytes for the tray
// icon. It is called by main before Wails starts.
func (a *App) EnableTray(iconBytes []byte) {
	a.trayMu.Lock()
	a.trayEnabled = true
	a.trayIcon = append([]byte(nil), iconBytes...)
	ctx := a.ctx
	a.trayMu.Unlock()
	if ctx != nil {
		a.startTray(ctx)
	}
}

// BeforeClose is called by Wails before the application window closes
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.mu.RLock()
	force := a.forceExit
	appCtx := a.ctx
	a.mu.RUnlock()
	if force {
		a.saveWindowState(ctx)
		return false
	}

	a.trayMu.Lock()
	trayEnabled := a.trayEnabled
	a.trayMu.Unlock()
	if trayEnabled && appCtx != nil {
		runtime.WindowHide(appCtx)
		return true
	}

	if !a.HasSettingsChanged() {
		a.saveWindowState(ctx)
		return false
	}

	// Notify frontend to display the custom in-app exit confirmation dialog
	if ctx != nil && a.ctx != nil {
		runtime.EventsEmit(ctx, "app:request-exit")
	}
	return true
}

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
	a.restoreWindowState(ctx)

	a.trayMu.Lock()
	trayEnabled := a.trayEnabled
	a.trayMu.Unlock()
	if trayEnabled {
		a.startTray(ctx)
	}
}

func (a *App) Shutdown(ctx context.Context) {
	a.stopTray()
	a.StopPolling()
	a.DisconnectAll()
	if a.history != nil {
		_ = a.history.Close()
	}
}

// GetSettings returns current application settings
func (a *App) GetSettings() AppSettings {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.settings
}

// SaveSettings updates internal settings
func (a *App) SaveSettings(s AppSettings) {
	s = normalizeSettings(s)
	a.mu.Lock()
	wasPolling := a.isPolling
	if !wasPolling {
		oldLinks := append([]PlcLinkSettings(nil), a.settings.PlcLinks...)
		a.settings = s
		a.mu.Unlock()

		for _, old := range oldLinks {
			current := findLink(s.PlcLinks, old.Name)
			if current == nil || old.IpAddress != current.IpAddress || old.Rack != current.Rack || old.Slot != current.Slot {
				_ = a.Disconnect(old.Name)
			}
		}
		return
	}
	a.mu.Unlock()

	a.StartPolling(s)
}

func findLink(links []PlcLinkSettings, name string) *PlcLinkSettings {
	for index := range links {
		if links[index].Name == name {
			return &links[index]
		}
	}
	return nil
}

// TestConnection tests connecting to a single PLC link
func (a *App) TestConnection(link PlcLinkSettings) error {
	handler := gos7.NewTCPClientHandler(link.IpAddress, link.Rack, link.Slot)
	handler.Timeout = 2 * time.Second
	handler.IdleTimeout = 2 * time.Second

	if err := handler.Connect(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer handler.Close()
	return nil
}

// Connect to a specific S7 PLC
func (a *App) Connect(linkName string, ip string, rack int, slot int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if conn, exists := a.plcs[linkName]; exists {
		conn.mu.Lock()
		connected := conn.IsConnected
		conn.mu.Unlock()
		if connected {
			return nil
		}
	}

	handler := gos7.NewTCPClientHandler(ip, rack, slot)
	handler.Timeout = 2 * time.Second
	handler.IdleTimeout = 2 * time.Second

	if err := handler.Connect(); err != nil {
		return fmt.Errorf("failed to connect to %s (%s): %w", linkName, ip, err)
	}

	client := gos7.NewClient(handler)
	a.plcs[linkName] = &PlcConnection{
		Client:      client,
		Handler:     handler,
		IsConnected: true,
	}
	return nil
}

// Disconnect from a PLC link
func (a *App) Disconnect(linkName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	conn, exists := a.plcs[linkName]
	if !exists {
		return nil
	}

	conn.mu.Lock()
	if !conn.IsConnected {
		conn.mu.Unlock()
		return nil
	}
	err := conn.Handler.Close()
	conn.IsConnected = false
	conn.mu.Unlock()
	delete(a.plcs, linkName)
	return err
}

func (a *App) DisconnectAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, conn := range a.plcs {
		conn.mu.Lock()
		if conn.IsConnected {
			_ = conn.Handler.Close()
			conn.IsConnected = false
		}
		conn.mu.Unlock()
	}
	a.plcs = make(map[string]*PlcConnection)
}

func (a *App) CheckStatus(linkName string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	conn, exists := a.plcs[linkName]
	if !exists {
		return false
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.IsConnected
}

// RecordSample persists a data point when the tag's configured storage
// interval has elapsed. A zero tag interval follows the global poll interval.
func (a *App) RecordSample(tagId string, timestampMs int64, value float64) {
	if a.history == nil {
		return
	}

	intervalMs := a.sampleIntervalMs(tagId)
	a.sampleMu.Lock()
	if a.isBoolTag(tagId) {
		if last, exists := a.lastBoolValue[tagId]; exists && last == value {
			a.sampleMu.Unlock()
			return
		}
		if _, initialized := a.lastBoolValue[tagId]; !initialized {
			if latest, exists, err := a.history.Latest(tagId); err == nil && exists {
				a.lastBoolValue[tagId] = latest.Value
				if latest.Value == value {
					a.sampleMu.Unlock()
					return
				}
			}
		}
	} else if last, exists := a.lastSampleAt[tagId]; exists && timestampMs < last+intervalMs {
		a.sampleMu.Unlock()
		return
	}

	err := a.history.Insert(storedSample{
		TagID:     tagId,
		Timestamp: timestampMs,
		Value:     value,
	})
	if err == nil {
		if a.isBoolTag(tagId) {
			a.lastBoolValue[tagId] = value
		} else {
			a.lastSampleAt[tagId] = timestampMs
		}
	}
	a.sampleMu.Unlock()
}

// GetHistoryRange returns sample points for requested tagIds within [startMs, endMs]
func (a *App) GetHistoryRange(tagIds []string, startMs, endMs int64) map[string][]SamplePoint {
	if a.history == nil {
		result := make(map[string][]SamplePoint, len(tagIds))
		for _, tagId := range tagIds {
			result[tagId] = []SamplePoint{}
		}
		return result
	}
	result, _ := a.history.GetRange(tagIds, startMs, endMs)
	return result
}

// ClearHistory clears all recorded samples from the local history database.
func (a *App) ClearHistory() {
	a.sampleMu.Lock()
	defer a.sampleMu.Unlock()
	if a.history != nil {
		_ = a.history.Clear()
	}
	a.lastSampleAt = make(map[string]int64)
	a.lastBoolValue = make(map[string]float64)
}

func (a *App) sampleIntervalMs(tagID string) int64 {
	a.mu.RLock()
	interval := a.settings.PollIntervalMs
	for _, tag := range a.settings.Tags {
		if tag.Id.String() == tagID {
			if tag.SamplingIntervalMs > 0 {
				interval = tag.SamplingIntervalMs
			}
			break
		}
	}
	a.mu.RUnlock()

	if interval < 10 {
		interval = 10
	}
	return int64(interval)
}

func (a *App) isBoolTag(tagID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, tag := range a.settings.Tags {
		if tag.Id.String() == tagID {
			return tag.DataType == DataTypeBool
		}
	}
	return false
}
