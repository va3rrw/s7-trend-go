package backend

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// File Operations

type AppState struct {
	LastSettingsFile string `json:"lastSettingsFile"`
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
	if state.LastSettingsFile != "" {
		if _, err := os.Stat(state.LastSettingsFile); err == nil {
			settingsData, err := os.ReadFile(state.LastSettingsFile)
			if err == nil {
				var settings AppSettings
				if err := json.Unmarshal(settingsData, &settings); err == nil {
					if settings.PollIntervalMs == 0 {
						settings.PollIntervalMs = 100
					}
					if settings.TimeWindowSeconds == 0 {
						settings.TimeWindowSeconds = 60
					}
					if settings.Interpolation == "" {
						settings.Interpolation = InterpolationLine
					}
					if len(settings.PlcLinks) == 0 {
						settings.PlcLinks = CreateDefaultPlcLinks()
					}
					if len(settings.YAxes) == 0 {
						settings.YAxes = CreateDefaultYAxes()
					}
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
	state := AppState{
		LastSettingsFile: lastFile,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(statePath, data, 0644)
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

// SaveCurrentSettings writes current in-memory settings to the last-used settings file path,
// or opens a save file dialog if no file path is set yet.
func (a *App) SaveCurrentSettings(ctx context.Context) error {
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
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
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
		Title:           title,
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
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
	if settings.PollIntervalMs == 0 {
		settings.PollIntervalMs = 100
	}
	if settings.TimeWindowSeconds == 0 {
		settings.TimeWindowSeconds = 60
	}
	if settings.Interpolation == "" {
		settings.Interpolation = InterpolationLine
	}
	if len(settings.PlcLinks) == 0 {
		settings.PlcLinks = CreateDefaultPlcLinks()
	}
	if len(settings.YAxes) == 0 {
		settings.YAxes = CreateDefaultYAxes()
	}

	a.mu.Lock()
	a.settings = settings
	a.lastSettingsPath = filePath
	a.savedSettingsJSON = a.serializeSettingsLocked()
	a.mu.Unlock()

	a.saveAppState(filePath)
	return &settings, nil
}

// ExportCSV streams all recorded sample history from backend ring buffers directly to disk
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

	a.historyMu.RLock()
	defer a.historyMu.RUnlock()

	tagMap := make(map[string]TagSettings)
	a.mu.RLock()
	for _, t := range a.settings.Tags {
		tagMap[t.Id.String()] = t
	}
	a.mu.RUnlock()

	for tagIdStr, rb := range a.history {
		tag, exists := tagMap[tagIdStr]
		tagName := tagIdStr
		tagAddr := ""
		if exists {
			tagName = tag.Name
			tagAddr = tag.Address
		}
		points := rb.GetAll()
		for _, pt := range points {
			tStr := time.UnixMilli(pt.Timestamp).Format(time.RFC3339Nano)
			if err := writer.Write([]string{
				tStr,
				tagName,
				tagAddr,
				strconv.FormatFloat(pt.Value, 'f', -1, 64),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
