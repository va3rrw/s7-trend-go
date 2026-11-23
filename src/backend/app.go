package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jeandeaual/go-locale"
	"github.com/robinson/gos7"
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
	host              DesktopHost
	plcs              map[string]*PlcConnection
	settings          AppSettings
	savedSettingsJSON string
	lastSettingsPath  string
	forceExit         bool
	mu                sync.RWMutex
	cancelPoll        func()
	isPolling         bool
	pollDone          chan struct{}

	history        *HistoryStore
	historyErr     error
	sampleMu       sync.Mutex
	lastSampleAt   map[string]int64
	lastBoolValue  map[string]float64
	windowWidth    int
	windowHeight   int
	windowX        int
	windowY        int
	windowPosSet   bool
	treePanelWidth int

	trayMu      sync.Mutex
	trayEnabled bool

	telemetryMu   sync.RWMutex
	telemetrySubs map[uint64]chan TelemetryFrame
	telemetrySeq  uint64

	eventMu   sync.RWMutex
	eventSubs map[uint64]chan AppEvent
	eventSeq  uint64

	webOpMu          sync.Mutex
	webMu            sync.Mutex
	webAssets        fs.FS
	webServerEnabled bool
	webServerHost    string
	webServerPort    int
	webServer        *http.Server
	webListener      net.Listener
	webRunning       bool
	webLastErr       string
	webConnCancel    context.CancelFunc
	webClients       atomic.Int64
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
		plcs:           make(map[string]*PlcConnection),
		settings:       CreateDefaultSettings(),
		history:        history,
		historyErr:     historyErr,
		lastSampleAt:   make(map[string]int64),
		lastBoolValue:  make(map[string]float64),
		treePanelWidth: defaultTreePanelWidth,
		telemetrySubs:  make(map[uint64]chan TelemetryFrame),
		eventSubs:      make(map[uint64]chan AppEvent),
		webServerHost:  defaultWebServerHost,
		webServerPort:  defaultWebServerPort,
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

func (a *App) getHost() DesktopHost {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.host
}

// QuitApp forces application exit without prompting
func (a *App) QuitApp() {
	a.mu.Lock()
	a.forceExit = true
	host := a.host
	a.mu.Unlock()
	a.saveWindowState()
	if host != nil {
		host.Quit()
	}
}

// IsPolling reports whether PLC polling is currently active.
func (a *App) IsPolling() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isPolling
}

func (a *App) plcConnectionCountsLocked() (connectedPlcs, totalPlcs, enabledTags int) {
	activeLinks := make(map[string]struct{})
	for _, tag := range a.settings.Tags {
		if tag.Enabled {
			enabledTags++
			if tag.PlcLink != "" {
				activeLinks[tag.PlcLink] = struct{}{}
			}
		}
	}
	if len(activeLinks) == 0 {
		for _, link := range a.settings.PlcLinks {
			if link.Name != "" {
				activeLinks[link.Name] = struct{}{}
			}
		}
	}
	for linkName := range activeLinks {
		if conn, ok := a.plcs[linkName]; ok && conn != nil {
			conn.mu.Lock()
			if conn.IsConnected {
				connectedPlcs++
			}
			conn.mu.Unlock()
		}
	}
	return connectedPlcs, len(activeLinks), enabledTags
}

// refreshTrayStatus pushes current polling state and connected PLC counts to the attached host.
func (a *App) refreshTrayStatus() {
	a.mu.RLock()
	host := a.host
	if host == nil {
		a.mu.RUnlock()
		return
	}
	isPolling := a.isPolling
	connectedPlcs, totalPlcs, _ := a.plcConnectionCountsLocked()
	a.mu.RUnlock()

	host.UpdateTrayStatus(isPolling, connectedPlcs, totalPlcs)
}

// ServiceShutdown cleans up web server, polling loops, PLC connections, subscribers, and SQLite history on Wails shutdown.
func (a *App) ServiceShutdown() error {
	a.shutdownWebServer()
	a.StopPolling()
	a.DisconnectAll()
	a.closeAllTelemetrySubscribers()
	a.closeAllEventSubscribers()
	if a.history != nil {
		_ = a.history.Close()
	}
	return nil
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
		oldSerialized := a.serializeSettingsLocked()
		a.settings = s
		changed := a.serializeSettingsLocked() != oldSerialized
		a.mu.Unlock()

		for _, old := range oldLinks {
			current := findLink(s.PlcLinks, old.Name)
			if current == nil || old.IpAddress != current.IpAddress || old.Rack != current.Rack || old.Slot != current.Slot {
				_ = a.Disconnect(old.Name)
			}
		}
		a.refreshTrayStatus()
		if changed {
			a.emitAppEvent("settings:updated", s)
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
	if conn, exists := a.plcs[linkName]; exists {
		conn.mu.Lock()
		connected := conn.IsConnected
		conn.mu.Unlock()
		if connected {
			a.mu.Unlock()
			return nil
		}
	}

	handler := gos7.NewTCPClientHandler(ip, rack, slot)
	handler.Timeout = 2 * time.Second
	handler.IdleTimeout = 2 * time.Second

	if err := handler.Connect(); err != nil {
		a.mu.Unlock()
		return fmt.Errorf("failed to connect to %s (%s): %w", linkName, ip, err)
	}

	client := gos7.NewClient(handler)
	a.plcs[linkName] = &PlcConnection{
		Client:      client,
		Handler:     handler,
		IsConnected: true,
	}
	a.mu.Unlock()
	a.refreshTrayStatus()
	return nil
}

// Disconnect from a PLC link
func (a *App) Disconnect(linkName string) error {
	a.mu.Lock()
	conn, exists := a.plcs[linkName]
	if !exists {
		a.mu.Unlock()
		return nil
	}

	conn.mu.Lock()
	if !conn.IsConnected {
		conn.mu.Unlock()
		a.mu.Unlock()
		return nil
	}
	err := conn.Handler.Close()
	conn.IsConnected = false
	conn.mu.Unlock()
	delete(a.plcs, linkName)
	a.mu.Unlock()

	a.refreshTrayStatus()
	return err
}

func (a *App) DisconnectAll() {
	a.mu.Lock()
	hadConnections := len(a.plcs) > 0
	for _, conn := range a.plcs {
		conn.mu.Lock()
		if conn.IsConnected {
			_ = conn.Handler.Close()
			conn.IsConnected = false
		}
		conn.mu.Unlock()
	}
	a.plcs = make(map[string]*PlcConnection)
	a.mu.Unlock()

	if hadConnections {
		a.refreshTrayStatus()
	}
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

// Flush waits for all queued history samples to be written to disk.
func (a *App) Flush() {
	if a.history != nil {
		a.history.Flush()
	}
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
		a.lastBoolValue[tagId] = value
	} else if last, exists := a.lastSampleAt[tagId]; exists && timestampMs < last+intervalMs {
		a.sampleMu.Unlock()
		return
	} else {
		a.lastSampleAt[tagId] = timestampMs
	}
	a.sampleMu.Unlock()

	_ = a.history.Insert(storedSample{
		TagID:     tagId,
		Timestamp: timestampMs,
		Value:     value,
	})
}

// recordPolledSample persists an acquired PLC sample. Bool tags store initial
// values and transitions; tags with an explicit Storage interval > 0 enforce
// that minimum gap; tags configured to follow the global poll interval (0)
// record every read without rejecting 1-2ms ticker/network jitter.
func (a *App) recordPolledSample(tag TagSettings, timestampMs int64, value float64) {
	if a.history == nil {
		return
	}

	tagID := tag.Id.String()
	a.sampleMu.Lock()
	if tag.DataType == DataTypeBool {
		if last, exists := a.lastBoolValue[tagID]; exists && last == value {
			a.sampleMu.Unlock()
			return
		}
		a.lastBoolValue[tagID] = value
	} else if tag.SamplingIntervalMs > 0 {
		if last, exists := a.lastSampleAt[tagID]; exists && timestampMs < last+int64(tag.SamplingIntervalMs) {
			a.sampleMu.Unlock()
			return
		}
		a.lastSampleAt[tagID] = timestampMs
	} else {
		a.lastSampleAt[tagID] = timestampMs
	}
	a.sampleMu.Unlock()

	_ = a.history.Insert(storedSample{
		TagID:     tagID,
		Timestamp: timestampMs,
		Value:     value,
	})
}

// GetHistoryRange returns sample points for requested tagIds within [startMs, endMs].
// Wails v3 binds the context parameter to the frontend CancellablePromise so
// panning, zooming, or clear operations cancel superseded queries cleanly.
func (a *App) GetHistoryRange(ctx context.Context, tagIds []string, startMs, endMs int64, maxPoints int) (map[string][]SamplePoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.history == nil {
		result := make(map[string][]SamplePoint, len(tagIds))
		for _, tagId := range tagIds {
			result[tagId] = []SamplePoint{}
		}
		return result, nil
	}
	if err := a.history.FlushContext(ctx); err != nil {
		return nil, err
	}
	return a.history.GetRangeContext(ctx, tagIds, startMs, endMs, maxPoints)
}

// ClearHistory clears all recorded samples from the local history database.
func (a *App) ClearHistory() {
	a.Flush()
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
