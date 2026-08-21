package backend

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCreateDefaultPlcLinks(t *testing.T) {
	links := CreateDefaultPlcLinks()
	if len(links) != 1 {
		t.Fatalf("expected 1 default PLC link, got %d", len(links))
	}

	if links[0].Name != "PLC1" {
		t.Errorf("links[0].Name = %s; expected PLC1", links[0].Name)
	}
	if links[0].IpAddress != "192.168.0.1" {
		t.Errorf("links[0].IpAddress = %s; expected 192.168.0.1", links[0].IpAddress)
	}
	if links[0].Rack != 0 {
		t.Errorf("links[0].Rack = %d; expected 0", links[0].Rack)
	}
	if links[0].Slot != 1 {
		t.Errorf("links[0].Slot = %d; expected 1", links[0].Slot)
	}
	if links[0].IsConnected {
		t.Errorf("links[0].IsConnected = true; expected false")
	}
}

func TestCreateDefaultYAxes(t *testing.T) {
	axes := CreateDefaultYAxes()
	if len(axes) != 4 {
		t.Fatalf("expected 4 default Y axes, got %d", len(axes))
	}

	expectedNames := []string{"Y-Axis 1", "Y-Axis 2", "Y-Axis 3", "Y-Axis 4"}
	for i, ax := range axes {
		if ax.Name != expectedNames[i] {
			t.Errorf("axes[%d].Name = %s; expected %s", i, ax.Name, expectedNames[i])
		}
		if ax.Minimum != 0 {
			t.Errorf("axes[%d].Minimum = %f; expected 0", i, ax.Minimum)
		}
		if ax.Maximum != 100 {
			t.Errorf("axes[%d].Maximum = %f; expected 100", i, ax.Maximum)
		}
		if !ax.AutoScale {
			t.Errorf("axes[%d].AutoScale = false; expected true", i)
		}
	}
}

func TestCreateDefaultSettings(t *testing.T) {
	s := CreateDefaultSettings()
	if s.PollIntervalMs != 100 {
		t.Errorf("expected PollIntervalMs 100, got %d", s.PollIntervalMs)
	}
	if s.TimeWindowSeconds != 60 {
		t.Errorf("expected TimeWindowSeconds 60, got %d", s.TimeWindowSeconds)
	}
	if s.Interpolation != InterpolationLine {
		t.Errorf("expected Interpolation Line, got %s", s.Interpolation)
	}
	if len(s.PlcLinks) != 1 {
		t.Errorf("expected 1 PlcLink, got %d", len(s.PlcLinks))
	}
	if len(s.Tags) != 0 {
		t.Errorf("expected 0 Tags, got %d", len(s.Tags))
	}
	if len(s.YAxes) != 4 {
		t.Errorf("expected 4 YAxes, got %d", len(s.YAxes))
	}
}

func TestModels_JSONSerialization(t *testing.T) {
	tagId := uuid.New()
	tag := TagSettings{
		Id:       tagId,
		Name:     "Motor_Speed",
		PlcLink:  "PLC1",
		Address:  "DB1.DBD0",
		DataType: DataTypeReal,
		YAxis:    "Y-Axis 1",
		Color:    "#FF0000",
		Enabled:  true,
	}

	settings := AppSettings{
		PollIntervalMs:    250,
		TimeWindowSeconds: 120,
		Interpolation:     InterpolationStep,
		PlcLinks:          CreateDefaultPlcLinks(),
		Tags:              []TagSettings{tag},
		YAxes:             CreateDefaultYAxes(),
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("failed to marshal AppSettings: %v", err)
	}

	var restored AppSettings
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal AppSettings: %v", err)
	}

	if restored.PollIntervalMs != 250 {
		t.Errorf("restored.PollIntervalMs = %d, expected 250", restored.PollIntervalMs)
	}
	if restored.TimeWindowSeconds != 120 {
		t.Errorf("restored.TimeWindowSeconds = %d, expected 120", restored.TimeWindowSeconds)
	}
	if restored.Interpolation != InterpolationStep {
		t.Errorf("restored.Interpolation = %s, expected Step", restored.Interpolation)
	}
	if len(restored.Tags) != 1 {
		t.Fatalf("restored tags length = %d, expected 1", len(restored.Tags))
	}
	if restored.Tags[0].Id != tagId {
		t.Errorf("restored tag ID = %v, expected %v", restored.Tags[0].Id, tagId)
	}
	if restored.Tags[0].Name != "Motor_Speed" {
		t.Errorf("restored tag Name = %s, expected Motor_Speed", restored.Tags[0].Name)
	}
	if restored.Tags[0].DataType != DataTypeReal {
		t.Errorf("restored tag DataType = %s, expected Real", restored.Tags[0].DataType)
	}

	// Test PollUpdate serialization
	val := 42.5
	update := PollUpdate{
		TagId:        tagId,
		Timestamp:    "2026-08-18T12:00:00Z",
		Value:        "42.500",
		NumericValue: &val,
		Quality:      "Good",
		Error:        "",
	}
	updateData, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal PollUpdate: %v", err)
	}
	var restoredUpdate PollUpdate
	if err := json.Unmarshal(updateData, &restoredUpdate); err != nil {
		t.Fatalf("failed to unmarshal PollUpdate: %v", err)
	}
	if restoredUpdate.Value != "42.500" || *restoredUpdate.NumericValue != 42.5 {
		t.Errorf("restored update mismatch: %+v", restoredUpdate)
	}

	// Test PollTiming serialization
	timing := PollTiming{
		PlcLink:          "PLC1",
		ActualIntervalMs: 85,
	}
	timingData, err := json.Marshal(timing)
	if err != nil {
		t.Fatalf("failed to marshal PollTiming: %v", err)
	}
	var restoredTiming PollTiming
	if err := json.Unmarshal(timingData, &restoredTiming); err != nil {
		t.Fatalf("failed to unmarshal PollTiming: %v", err)
	}
	if restoredTiming.PlcLink != "PLC1" || restoredTiming.ActualIntervalMs != 85 {
		t.Errorf("restored timing mismatch: %+v", restoredTiming)
	}

	// Test AppSettings serialization
	appSettings := CreateDefaultSettings()
	appSettings.Tags = []TagSettings{tag}
	cfgData, err := json.Marshal(appSettings)
	if err != nil {
		t.Fatalf("failed to marshal AppSettings: %v", err)
	}
	var restoredCfg AppSettings
	if err := json.Unmarshal(cfgData, &restoredCfg); err != nil {
		t.Fatalf("failed to unmarshal AppSettings: %v", err)
	}
	if len(restoredCfg.PlcLinks) != 1 || len(restoredCfg.Tags) != 1 || len(restoredCfg.YAxes) != 4 {
		t.Errorf("restored AppSettings mismatch: %+v", restoredCfg)
	}
}
