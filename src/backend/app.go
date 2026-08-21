package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	mu                sync.RWMutex
	cancelPoll        context.CancelFunc
	isPolling         bool
	pollDone          chan struct{}

	historyMu sync.RWMutex
	history   map[string]*TagRingBuffer
}

func NewApp() *App {
	app := &App{
		plcs:     make(map[string]*PlcConnection),
		settings: CreateDefaultSettings(),
		history:  make(map[string]*TagRingBuffer),
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

// BeforeClose is called by Wails before the application window closes
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	if !a.HasSettingsChanged() {
		return false
	}

	lang := a.GetSystemLanguage()
	isZh := strings.HasPrefix(strings.ToLower(lang), "zh")

	title := "Save Settings"
	message := "Settings have been modified. Do you want to save changes before exiting?"
	saveBtn := "Save"
	dontSaveBtn := "Don't Save"
	cancelBtn := "Cancel"

	if isZh {
		title = "保存设置"
		message = "配置已被修改。是否在退出前保存更改？"
		saveBtn = "保存"
		dontSaveBtn = "不保存"
		cancelBtn = "取消"
	}

	res, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         title,
		Message:       message,
		Buttons:       []string{saveBtn, dontSaveBtn, cancelBtn},
		DefaultButton: saveBtn,
		CancelButton:  cancelBtn,
	})
	if err != nil {
		return false
	}

	if res == saveBtn || res == "Save" || res == "Yes" {
		if err := a.SaveCurrentSettings(ctx); err != nil {
			return true // cancel exit if saving failed or was cancelled
		}
		return false
	}
	if res == dontSaveBtn || res == "Don't Save" || res == "No" {
		return false
	}
	if res == cancelBtn || res == "Cancel" {
		return true
	}
	return false
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Shutdown(ctx context.Context) {
	a.StopPolling()
	a.DisconnectAll()
}

// GetSettings returns current application settings
func (a *App) GetSettings() AppSettings {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.settings
}

// SaveSettings updates internal settings
func (a *App) SaveSettings(s AppSettings) {
	a.mu.RLock()
	wasPolling := a.isPolling
	a.mu.RUnlock()

	if wasPolling {
		a.StartPolling(s)
		return
	}

	a.mu.Lock()
	oldLinks := append([]PlcLinkSettings(nil), a.settings.PlcLinks...)
	a.settings = s
	a.mu.Unlock()

	for _, old := range oldLinks {
		current := findLink(s.PlcLinks, old.Name)
		if current == nil || old.IpAddress != current.IpAddress || old.Rack != current.Rack || old.Slot != current.Slot {
			_ = a.Disconnect(old.Name)
		}
	}
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

	for linkName, conn := range a.plcs {
		conn.mu.Lock()
		if conn.IsConnected {
			conn.Handler.Close()
		}
		conn.IsConnected = false
		conn.mu.Unlock()
		delete(a.plcs, linkName)
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

// RecordSample stores a data point into the tag's backend ring buffer
func (a *App) RecordSample(tagId string, timestampMs int64, value float64) {
	a.historyMu.Lock()
	rb, exists := a.history[tagId]
	if !exists {
		rb = NewTagRingBuffer(500000)
		a.history[tagId] = rb
	}
	a.historyMu.Unlock()

	rb.Push(timestampMs, value)
}

// GetHistoryRange returns sample points for requested tagIds within [startMs, endMs]
func (a *App) GetHistoryRange(tagIds []string, startMs, endMs int64) map[string][]SamplePoint {
	result := make(map[string][]SamplePoint)
	a.historyMu.RLock()
	defer a.historyMu.RUnlock()

	for _, tagId := range tagIds {
		if rb, exists := a.history[tagId]; exists {
			result[tagId] = rb.GetRange(startMs, endMs)
		} else {
			result[tagId] = []SamplePoint{}
		}
	}
	return result
}

// ClearHistory clears all recorded samples from backend ring buffers
func (a *App) ClearHistory() {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()

	for _, rb := range a.history {
		rb.Clear()
	}
}

