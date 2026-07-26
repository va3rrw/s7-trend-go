package backend

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

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

func (a *App) ExportCSV(records []SampleRecord, title string) error {
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

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"Timestamp", "Tag", "Address", "Value"}); err != nil {
		return err
	}
	for _, r := range records {
		if err := writer.Write([]string{
			r.Timestamp,
			r.TagName,
			r.Address,
			fmt.Sprintf("%f", r.Value),
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}
