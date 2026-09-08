package backend

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestHistoryStoreCreatesAndReopensFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "history.db")
	app := NewAppWithHistoryPath(dbPath)
	if app.historyErr != nil {
		t.Fatalf("opening history database: %v", app.historyErr)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected history database to be created automatically: %v", err)
	}
	app.RecordSample("tag-1", 1000, 12.5)
	app.Shutdown(context.Background())

	reopened := NewAppWithHistoryPath(dbPath)
	defer reopened.Shutdown(context.Background())
	points := reopened.GetHistoryRange([]string{"tag-1"}, 0, 2000)["tag-1"]
	if len(points) != 1 || points[0].Timestamp != 1000 || points[0].Value != 12.5 {
		t.Fatalf("expected persisted point, got %+v", points)
	}
}

func TestAppSamplingIntervalAndBooleanChanges(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history.db")
	app := NewAppWithHistoryPath(dbPath)
	defer app.Shutdown(context.Background())

	numericID := uuid.New()
	boolID := uuid.New()
	settings := CreateDefaultSettings()
	settings.PollIntervalMs = 1000
	settings.Tags = []TagSettings{
		{Id: numericID, DataType: DataTypeReal, SamplingIntervalMs: 100},
		{Id: boolID, DataType: DataTypeBool},
	}
	app.SaveSettings(settings)
	if got := app.GetSettings().Tags[0].SamplingIntervalMs; got != 1000 {
		t.Fatalf("expected tag interval to follow the 1000 ms poll interval, got %d", got)
	}

	app.RecordSample(numericID.String(), 1000, 1)
	app.RecordSample(numericID.String(), 1500, 2)
	app.RecordSample(numericID.String(), 2000, 3)
	app.RecordSample(boolID.String(), 1000, 0)
	app.RecordSample(boolID.String(), 1100, 0)
	app.RecordSample(boolID.String(), 1200, 1)
	app.RecordSample(boolID.String(), 1300, 1)

	points := app.GetHistoryRange([]string{numericID.String(), boolID.String()}, 0, 2000)
	if got := points[numericID.String()]; len(got) != 2 || got[1].Value != 3 {
		t.Errorf("expected two numeric points at the configured interval, got %+v", got)
	}
	if got := points[boolID.String()]; len(got) != 2 || got[0].Value != 0 || got[1].Value != 1 {
		t.Errorf("expected only boolean initial state and transition, got %+v", got)
	}
}
