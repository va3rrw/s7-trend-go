package backend

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// File Operations

func (a *App) SavePlcTagSettings(links []PlcLinkSettings, tags []TagSettings, title string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "plc_tags.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		return err
	}
	cfg := PlcTagConfig{
		PlcLinks: links,
		Tags:     tags,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (a *App) LoadPlcTagSettings(title string) (*PlcTagConfig, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var cfg PlcTagConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Fallback: check if it's full AppSettings file
		var fullSettings AppSettings
		if err2 := json.Unmarshal(data, &fullSettings); err2 == nil && len(fullSettings.PlcLinks) > 0 {
			return &PlcTagConfig{
				PlcLinks: fullSettings.PlcLinks,
				Tags:     fullSettings.Tags,
			}, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (a *App) SaveSettingsFile(settings AppSettings, title string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "s7-trend-settings.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (a *App) LoadSettingsFile(title string) (*AppSettings, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
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
