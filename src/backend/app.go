package backend

import (
	"context"
	"fmt"
	"sync"
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
	ctx        context.Context
	plcs       map[string]*PlcConnection
	settings   AppSettings
	mu         sync.RWMutex
	cancelPoll context.CancelFunc
	isPolling  bool
	pollDone   chan struct{}
}

func NewApp() *App {
	return &App{
		plcs:     make(map[string]*PlcConnection),
		settings: CreateDefaultSettings(),
	}
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
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

// SaveSettings updates internal settings
func (a *App) SaveSettings(s AppSettings) {
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
