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

func (a *App) SavePlcLinks(links []PlcLinkSettings, title string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "plcs.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		return err
	}
	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (a *App) LoadPlcLinks(title string) ([]PlcLinkSettings, error) {
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
	var links []PlcLinkSettings
	if err := json.Unmarshal(data, &links); err != nil {
		return nil, err
	}
	return links, nil
}

func (a *App) SaveTags(tags []TagSettings, title string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "tags.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}},
	})
	if err != nil || filePath == "" {
		return err
	}
	data, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (a *App) LoadTags(title string) ([]TagSettings, error) {
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
	var tags []TagSettings
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, err
	}
	return tags, nil
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
