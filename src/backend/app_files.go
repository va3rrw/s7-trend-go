package backend

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// File Operations

type AppState struct {
	LastSettingsFile  string `json:"lastSettingsFile"`
	WindowWidth       int    `json:"windowWidth,omitempty"`
	WindowHeight      int    `json:"windowHeight,omitempty"`
	WindowX           int    `json:"windowX,omitempty"`
	WindowY           int    `json:"windowY,omitempty"`
	WindowPositionSet bool   `json:"windowPositionSet,omitempty"`
}

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
	a.mu.Unlock()
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
	}
	a.mu.RUnlock()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(statePath, data, 0644)
}

func (a *App) restoreWindowState(ctx context.Context) {
	if !hasWailsFrontend(ctx) {
		return
	}

	a.mu.RLock()
	width := a.windowWidth
	height := a.windowHeight
	x := a.windowX
	y := a.windowY
	positionSet := a.windowPosSet
	a.mu.RUnlock()

	if width <= 0 || height <= 0 {
		return
	}
	runtime.WindowSetSize(ctx, width, height)
	if positionSet {
		runtime.WindowSetPosition(ctx, x, y)
	}
}

func (a *App) saveWindowState(ctx context.Context) {
	if !hasWailsFrontend(ctx) {
		return
	}

	a.mu.RLock()
	lastFile := a.lastSettingsPath
	a.mu.RUnlock()

	width, height := runtime.WindowGetSize(ctx)
	x, y := runtime.WindowGetPosition(ctx)
	if width <= 0 || height <= 0 {
		return
	}

	a.mu.Lock()
	a.windowWidth = width
	a.windowHeight = height
	a.windowX = x
	a.windowY = y
	a.windowPosSet = true
	a.mu.Unlock()
	a.saveAppState(lastFile)
}

func hasWailsFrontend(ctx context.Context) bool {
	return ctx != nil && ctx.Value("frontend") != nil
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

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultDirectory: defaultDir,
		Title:            title,
		DefaultFilename:  defaultFilename,
		Filters:          []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		if filePath == "" {
			return fmt.Errorf("cancelled")
		}
		return err
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

	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: defaultDir,
		Title:            title,
		Filters:          []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		if filePath == "" {
			return nil, fmt.Errorf("cancelled")
		}
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	settings = normalizeSettings(settings)

	a.mu.Lock()
	a.settings = settings
	a.lastSettingsPath = filePath
	a.savedSettingsJSON = a.serializeSettingsLocked()
	a.mu.Unlock()

	a.saveAppState(filePath)
	return &settings, nil
}

// ExportCSV streams all recorded sample history from SQLite directly to disk.
func (a *App) ExportCSV(title string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "trend.csv",
		Filters:         []runtime.FileFilter{{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"}},
	})
	if err != nil || filePath == "" {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bufWriter := bufio.NewWriterSize(file, 64*1024)
	writer := csv.NewWriter(bufWriter)
	defer func() {
		writer.Flush()
		bufWriter.Flush()
	}()

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
	return a.history.IterateAll(func(sample storedSample) error {
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
	})
}
