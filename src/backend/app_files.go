package backend

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"
)

// File Operations

type AppState struct {
	LastSettingsFile  string `json:"lastSettingsFile"`
	WindowWidth       int    `json:"windowWidth,omitempty"`
	WindowHeight      int    `json:"windowHeight,omitempty"`
	WindowX           int    `json:"windowX,omitempty"`
	WindowY           int    `json:"windowY,omitempty"`
	WindowPositionSet bool   `json:"windowPositionSet,omitempty"`
	TreePanelWidth    int    `json:"treePanelWidth,omitempty"`
	WebServerEnabled  bool   `json:"webServerEnabled,omitempty"`
	WebServerHost     string `json:"webServerHost,omitempty"`
	WebServerPort     int    `json:"webServerPort,omitempty"`
}

const (
	defaultTreePanelWidth = 250
	minTreePanelWidth     = 220
	maxTreePanelWidth     = 420
)

func getAppStateFilePath() string {
	return filepath.Join(getDefaultSettingsDir(), "app_state.json")
}

func (a *App) loadAppState() {
	statePath := getAppStateFilePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return
	}
	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	a.mu.Lock()
	a.windowWidth = state.WindowWidth
	a.windowHeight = state.WindowHeight
	a.windowX = state.WindowX
	a.windowY = state.WindowY
	a.windowPosSet = state.WindowPositionSet
	if state.TreePanelWidth >= minTreePanelWidth && state.TreePanelWidth <= maxTreePanelWidth {
		a.treePanelWidth = state.TreePanelWidth
	}
	a.mu.Unlock()

	a.webMu.Lock()
	a.webServerEnabled = state.WebServerEnabled
	if trimmedHost := strings.TrimSpace(state.WebServerHost); trimmedHost != "" {
		a.webServerHost = trimmedHost
	}
	if state.WebServerPort >= minWebServerPort && state.WebServerPort <= maxWebServerPort {
		a.webServerPort = state.WebServerPort
	}
	a.webMu.Unlock()
	if state.LastSettingsFile != "" {
		if _, err := os.Stat(state.LastSettingsFile); err == nil {
			settingsData, err := os.ReadFile(state.LastSettingsFile)
			if err == nil {
				var settings AppSettings
				if err := json.Unmarshal(settingsData, &settings); err == nil {
					settings = normalizeSettings(settings)
					a.mu.Lock()
					a.settings = settings
					a.lastSettingsPath = state.LastSettingsFile
					a.savedSettingsJSON = a.serializeSettingsLocked()
					a.mu.Unlock()
				}
			}
		}
	}
}

func (a *App) saveAppState(lastFile string) {
	statePath := getAppStateFilePath()
	a.mu.RLock()
	state := AppState{
		LastSettingsFile:  lastFile,
		WindowWidth:       a.windowWidth,
		WindowHeight:      a.windowHeight,
		WindowX:           a.windowX,
		WindowY:           a.windowY,
		WindowPositionSet: a.windowPosSet,
		TreePanelWidth:    a.treePanelWidth,
	}
	a.mu.RUnlock()

	a.webMu.Lock()
	state.WebServerEnabled = a.webServerEnabled
	state.WebServerHost = a.webServerHost
	state.WebServerPort = a.webServerPort
	a.webMu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(statePath, data, 0644)
}

// GetTreePanelWidth returns the restored PLC/tag tree width in pixels.
func (a *App) GetTreePanelWidth() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.treePanelWidth < minTreePanelWidth || a.treePanelWidth > maxTreePanelWidth {
		return defaultTreePanelWidth
	}
	return a.treePanelWidth
}

// SetTreePanelWidth updates the PLC/tag tree width. It is persisted with the
// rest of the app state when the application closes.
func (a *App) SetTreePanelWidth(width int) {
	if width < minTreePanelWidth {
		width = minTreePanelWidth
	} else if width > maxTreePanelWidth {
		width = maxTreePanelWidth
	}
	a.mu.Lock()
	a.treePanelWidth = width
	a.mu.Unlock()
}

func (a *App) saveWindowState() {
	host := a.getHost()
	if host == nil {
		return
	}

	a.mu.RLock()
	lastFile := a.lastSettingsPath
	a.mu.RUnlock()

	if width, height, x, y, ok := host.GetWindowBounds(); ok && width > 0 && height > 0 {
		a.mu.Lock()
		a.windowWidth = width
		a.windowHeight = height
		a.windowX = x
		a.windowY = y
		a.windowPosSet = true
		a.mu.Unlock()
	}
	a.saveAppState(lastFile)
}

func getDefaultSettingsDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}
	dir := filepath.Join(configDir, "s7-trend-go")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// OpenStorageFolder opens the directory containing app_state.json and history.db.
func (a *App) OpenStorageFolder() error {
	dir := getDefaultSettingsDir()
	var command string
	switch goruntime.GOOS {
	case "windows":
		command = "explorer.exe"
	case "darwin":
		command = "open"
	default:
		command = "xdg-open"
	}
	return exec.Command(command, dir).Start()
}

// SaveCurrentSettings writes current in-memory settings to the last-used settings file path,
// or opens a save file dialog if no file path is set yet.
func (a *App) SaveCurrentSettings() error {
	a.mu.RLock()
	lastPath := a.lastSettingsPath
	settings := a.settings
	a.mu.RUnlock()

	if lastPath != "" {
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(lastPath, data, 0644); err != nil {
			return err
		}
		a.mu.Lock()
		a.savedSettingsJSON = a.serializeSettingsLocked()
		a.mu.Unlock()
		return nil
	}

	return a.SaveSettingsFile(settings, "Save Settings")
}

func (a *App) SaveSettingsFile(settings AppSettings, title string) error {
	settings = normalizeSettings(settings)
	defaultDir := getDefaultSettingsDir()
	defaultFilename := "s7-trend-settings.json"

	a.mu.RLock()
	lastPath := a.lastSettingsPath
	a.mu.RUnlock()

	if lastPath != "" {
		defaultFilename = filepath.Base(lastPath)
		defaultDir = filepath.Dir(lastPath)
	}

	host := a.getHost()
	if host == nil {
		return fmt.Errorf("desktop runtime unavailable")
	}

	filePath, err := host.SaveFileDialog(title, defaultDir, defaultFilename, "JSON Files (*.json)", "*.json")
	if err != nil {
		return err
	}
	if filePath == "" {
		return fmt.Errorf("cancelled")
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return err
	}

	a.mu.Lock()
	a.settings = settings
	a.lastSettingsPath = filePath
	a.savedSettingsJSON = a.serializeSettingsLocked()
	a.mu.Unlock()

	a.saveAppState(filePath)
	return nil
}

func (a *App) LoadSettingsFile(title string) (*AppSettings, error) {
	defaultDir := getDefaultSettingsDir()

	a.mu.RLock()
	lastPath := a.lastSettingsPath
	a.mu.RUnlock()

	if lastPath != "" {
		defaultDir = filepath.Dir(lastPath)
	}

	host := a.getHost()
	if host == nil {
		return nil, fmt.Errorf("desktop runtime unavailable")
	}

	filePath, err := host.OpenFileDialog(title, defaultDir, "JSON Files (*.json)", "*.json")
	if err != nil {
		return nil, err
	}
	if filePath == "" {
		return nil, fmt.Errorf("cancelled")
	}
	settings, err := a.loadSettingsFromFilePath(filePath)
	if err == nil && settings != nil {
		a.emitAppEvent("settings:loaded", *settings)
	}
	return settings, err
}

func parseSettingsJSON(data []byte) (AppSettings, error) {
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("{")) {
		return AppSettings{}, fmt.Errorf("settings file must contain a JSON object")
	}
	var settings AppSettings
	if err := json.Unmarshal(trimmed, &settings); err != nil {
		return AppSettings{}, err
	}
	return normalizeSettings(settings), nil
}

func (a *App) loadSettingsFromFilePath(filePath string) (*AppSettings, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	settings, err := parseSettingsJSON(data)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.settings = settings
	a.lastSettingsPath = filePath
	a.savedSettingsJSON = a.serializeSettingsLocked()
	a.mu.Unlock()

	a.saveAppState(filePath)
	return &settings, nil
}

// ImportSettingsJSON parses and applies raw JSON settings uploaded from a web
// client, restarts active polling if running, and broadcasts the update.
func (a *App) ImportSettingsJSON(rawJSON string) (*AppSettings, error) {
	settings, err := parseSettingsJSON([]byte(rawJSON))
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.settings = settings
	a.savedSettingsJSON = a.serializeSettingsLocked()
	lastPath := a.lastSettingsPath
	a.mu.Unlock()

	if lastPath != "" {
		a.saveAppState(lastPath)
	}

	if a.IsPolling() {
		a.StartPolling(settings)
	} else {
		a.refreshTrayStatus()
	}

	a.emitAppEvent("settings:loaded", settings)
	return &settings, nil
}

// FindSettingsJSONArg inspects CLI arguments for an existing .json settings
// file path, resolving relative paths against workingDir when provided.
func FindSettingsJSONArg(args []string, workingDir string) string {
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" || strings.HasPrefix(trimmed, "-") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(trimmed), ".json") {
			continue
		}
		candidate := trimmed
		if !filepath.IsAbs(candidate) && workingDir != "" {
			candidate = filepath.Join(workingDir, candidate)
		}
		if abs, err := filepath.Abs(candidate); err == nil {
			candidate = abs
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// ExportCSV streams all recorded sample history from SQLite directly to disk.
func (a *App) ExportCSV(title string) error {
	host := a.getHost()
	if host == nil {
		return fmt.Errorf("desktop runtime unavailable")
	}

	filePath, err := host.SaveFileDialog(title, "", "trend.csv", "CSV Files (*.csv)", "*.csv")
	if err != nil || filePath == "" {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	if err := WriteCSV(a, file); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// WriteCSV streams all recorded SQLite samples formatted as CSV to w.
func WriteCSV(a *App, w io.Writer) error {
	a.Flush()

	bufWriter := bufio.NewWriterSize(w, 64*1024)
	writer := csv.NewWriter(bufWriter)

	if err := writer.Write([]string{"Timestamp", "Tag", "Address", "Value"}); err != nil {
		return err
	}

	tagMap := make(map[string]TagSettings)
	a.mu.RLock()
	for _, t := range a.settings.Tags {
		tagMap[t.Id.String()] = t
	}
	a.mu.RUnlock()

	if a.history == nil {
		return fmt.Errorf("history database is not available")
	}
	if err := a.history.IterateAll(func(sample storedSample) error {
		tag, exists := tagMap[sample.TagID]
		tagName := sample.TagID
		tagAddr := ""
		if exists {
			tagName = tag.Name
			tagAddr = tag.Address
		}
		return writer.Write([]string{
			time.UnixMilli(sample.Timestamp).Format(time.RFC3339Nano),
			tagName,
			tagAddr,
			strconv.FormatFloat(sample.Value, 'f', -1, 64),
		})
	}); err != nil {
		return err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return bufWriter.Flush()
}
