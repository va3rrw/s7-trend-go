package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	points := reopened.GetHistoryRange([]string{"tag-1"}, 0, 2000, 0)["tag-1"]
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
	app.Flush()

	points := app.GetHistoryRange([]string{numericID.String(), boolID.String()}, 0, 2000, 0)
	if got := points[numericID.String()]; len(got) != 2 || got[1].Value != 3 {
		t.Errorf("expected two numeric points at the configured interval, got %+v", got)
	}
	if got := points[boolID.String()]; len(got) != 2 || got[0].Value != 0 || got[1].Value != 1 {
		t.Errorf("expected only boolean initial state and transition, got %+v", got)
	}
}

func TestHistoryStoreDownsampling(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history_downsample.db")
	store, err := OpenHistoryStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// 1. Raw passthrough for small sets: 30 points inserted
	for i := int64(0); i < 30; i++ {
		err := store.Insert(storedSample{
			TagID:     "small-tag",
			Timestamp: 1000 + i*100,
			Value:     float64(i),
		})
		if err != nil {
			t.Fatalf("insert small: %v", err)
		}
	}
	store.Flush()

	// Request with maxPoints=500. Since raw count (30) <= maxPoints, raw points are passed through.
	rawResult, err := store.GetRange([]string{"small-tag"}, 1000, 4000, 500)
	if err != nil {
		t.Fatalf("get small range: %v", err)
	}
	if pts := rawResult["small-tag"]; len(pts) != 30 {
		t.Fatalf("expected raw passthrough for small set to return all 30 points, got %d", len(pts))
	}

	// 2. Point count bounded & extremes preserved on large dataset
	// Insert 2000 points into "big-tag" spanning [10000, 210000]
	// With an extreme peak and extreme valley embedded in the middle
	const numPoints = 2000
	const extremePeakTime = 50000
	const extremeValleyTime = 50500

	for i := int64(0); i < numPoints; i++ {
		ts := 10000 + i*100
		val := float64(i % 50)
		if ts == extremePeakTime {
			val = 9999.0 // extreme peak
		} else if ts == extremeValleyTime {
			val = -9999.0 // extreme valley
		}
		if err := store.Insert(storedSample{
			TagID:     "big-tag",
			Timestamp: ts,
			Value:     val,
		}); err != nil {
			t.Fatalf("insert big: %v", err)
		}
	}
	store.Flush()

	// Request with maxPoints = 200
	downsampled, err := store.GetRange([]string{"big-tag"}, 10000, 210000, 200)
	if err != nil {
		t.Fatalf("get downsampled range: %v", err)
	}
	pts := downsampled["big-tag"]
	if len(pts) == 0 {
		t.Fatalf("expected points, got none")
	}
	if len(pts) > 200 {
		t.Fatalf("expected point count to be bounded by maxPoints=200, got %d", len(pts))
	}

	// Verify extreme peak and valley are preserved with exact timestamps
	var foundPeak, foundValley bool
	var prevTs int64 = -1
	for _, p := range pts {
		if p.Timestamp <= prevTs {
			t.Errorf("expected strictly ascending timestamps, got %d after %d", p.Timestamp, prevTs)
		}
		prevTs = p.Timestamp
		if p.Timestamp == extremePeakTime && p.Value == 9999.0 {
			foundPeak = true
		}
		if p.Timestamp == extremeValleyTime && p.Value == -9999.0 {
			foundValley = true
		}
	}
	if !foundPeak {
		t.Errorf("expected downsampling to preserve extreme peak 9999.0 at ts=%d", extremePeakTime)
	}
	if !foundValley {
		t.Errorf("expected downsampling to preserve extreme valley -9999.0 at ts=%d", extremeValleyTime)
	}

	// 3. Clamping: maxPoints=10 clamps to 100 (returns <= 100 points)
	clampedResult, err := store.GetRange([]string{"big-tag"}, 10000, 210000, 10)
	if err != nil {
		t.Fatalf("get clamped range: %v", err)
	}
	if len(clampedResult["big-tag"]) > 100 {
		t.Fatalf("expected clamped maxPoints to limit points to <= 100, got %d", len(clampedResult["big-tag"]))
	}

	// 4. maxPoints <= 0 returns all raw points
	allRawResult, err := store.GetRange([]string{"big-tag"}, 10000, 210000, 0)
	if err != nil {
		t.Fatalf("get all raw range: %v", err)
	}
	if len(allRawResult["big-tag"]) != numPoints {
		t.Fatalf("expected maxPoints<=0 to return all %d raw points, got %d", numPoints, len(allRawResult["big-tag"]))
	}
}

func TestHistoryStoreTierSelectionAndExtremes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history_tiers.db")
	store, err := OpenHistoryStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// Insert samples over 2 hours (7,200,000 ms) with known extremes
	const startMs = int64(1000000)
	const endMs = startMs + 7200000
	const peakTs = startMs + 3600000
	const valleyTs = startMs + 3605000

	for ts := startMs; ts < endMs; ts += 500 {
		val := float64((ts / 1000) % 50)
		if ts == peakTs {
			val = 8888.0
		} else if ts == valleyTs {
			val = -8888.0
		}
		_ = store.Insert(storedSample{
			TagID:     "tag-tier",
			Timestamp: ts,
			Value:     val,
		})
	}
	store.Flush()

	// 1. Small window: 10s window (span = 10,000, maxPoints = 500 -> b = 10000/250 = 40 < 1000) -> raw query
	resSmall, err := store.GetRange([]string{"tag-tier"}, peakTs-2000, peakTs+2000, 500)
	if err != nil {
		t.Fatalf("small window GetRange: %v", err)
	}
	if len(resSmall["tag-tier"]) == 0 {
		t.Fatalf("expected points for small window")
	}

	// 2. Large window: 2 hours (span = 7,200,000, maxPoints = 200 -> b = 7200000 / 100 = 72,000 >= 60,000) -> 60s rollup tier
	resLarge, err := store.GetRange([]string{"tag-tier"}, startMs, endMs, 200)
	if err != nil {
		t.Fatalf("large window GetRange: %v", err)
	}
	pts := resLarge["tag-tier"]
	if len(pts) == 0 || len(pts) > 200 {
		t.Fatalf("expected bounded points (<= 200), got %d", len(pts))
	}

	// Verify extreme peak and valley are preserved in rollup
	var hasPeak, hasValley bool
	for _, p := range pts {
		if p.Timestamp == peakTs && p.Value == 8888.0 {
			hasPeak = true
		}
		if p.Timestamp == valleyTs && p.Value == -8888.0 {
			hasValley = true
		}
	}
	if !hasPeak {
		t.Errorf("expected extreme peak 8888.0 to be preserved in rollup query")
	}
	if !hasValley {
		t.Errorf("expected extreme valley -8888.0 to be preserved in rollup query")
	}
}

func TestHistoryStoreBackfillMigration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history_migration.db")

	// Create older schema DB without samples_rollup table
	oldDB, err := OpenHistoryStore(dbPath)
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	// Drop rollup table to simulate pre-migration state
	_, err = oldDB.db.Exec("DROP TABLE IF EXISTS samples_rollup")
	if err != nil {
		t.Fatalf("drop rollup: %v", err)
	}
	// Insert raw samples directly into samples table
	for i := int64(0); i < 500; i++ {
		_, err = oldDB.db.Exec(
			"INSERT INTO samples(tag_id, timestamp_ms, value) VALUES (?, ?, ?)",
			"mig-tag",
			100000+i*200,
			float64(i%10),
		)
		if err != nil {
			t.Fatalf("insert raw: %v", err)
		}
	}
	_ = oldDB.Close()

	// Reopen with OpenHistoryStore — should automatically backfill samples_rollup
	reopened, err := OpenHistoryStore(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()

	var rollupCount int
	err = reopened.readDB.QueryRow("SELECT COUNT(*) FROM samples_rollup WHERE tag_id = 'mig-tag'").Scan(&rollupCount)
	if err != nil {
		t.Fatalf("query rollup count: %v", err)
	}
	if rollupCount == 0 {
		t.Fatalf("expected backfill migration to populate samples_rollup, got 0 rows")
	}
}

func TestHistoryStoreClear(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history_clear.db")
	store, err := OpenHistoryStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for i := int64(0); i < 100; i++ {
		_ = store.Insert(storedSample{
			TagID:     "clear-tag",
			Timestamp: 1000 + i*100,
			Value:     float64(i),
		})
	}
	store.Flush()

	// Verify data exists
	res, err := store.GetRange([]string{"clear-tag"}, 0, 20000, 100)
	if err != nil || len(res["clear-tag"]) == 0 {
		t.Fatalf("expected data before clear")
	}

	// Clear both tables
	if err := store.Clear(); err != nil {
		t.Fatalf("clear error: %v", err)
	}

	var samplesCount, rollupCount int
	_ = store.readDB.QueryRow("SELECT COUNT(*) FROM samples").Scan(&samplesCount)
	_ = store.readDB.QueryRow("SELECT COUNT(*) FROM samples_rollup").Scan(&rollupCount)

	if samplesCount != 0 || rollupCount != 0 {
		t.Fatalf("expected 0 rows in both tables after clear, got samples=%d rollup=%d", samplesCount, rollupCount)
	}
}

func TestGetRange_24h_12Tags_Benchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 24h benchmark in short mode")
	}

	dbPath := filepath.Join(t.TempDir(), "history_24h_bench.db")
	store, err := OpenHistoryStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// 12 tags, 24h = 86,400,000 ms. At 200ms interval = 432,000 samples per tag.
	// In the rollup table at 60s tier: 1440 buckets per tag.
	// Populate samples and rollups in a single transaction
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	_, _ = tx.Exec("PRAGMA synchronous = OFF")

	rollupStmt, err := tx.Prepare(`
INSERT INTO samples_rollup(tag_id, res_ms, bucket_ms, min_t, min_v, max_t, max_v)
VALUES (?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		t.Fatalf("prepare rollup: %v", err)
	}

	tagIDs := make([]string, 12)
	for i := 0; i < 12; i++ {
		tagIDs[i] = fmt.Sprintf("bench-tag-%d", i)
	}

	// 24 hours = 1,440 buckets at 60s resolution
	const totalBuckets = 1440
	const res60s = int64(60000)
	for _, tagID := range tagIDs {
		for b := int64(0); b < totalBuckets; b++ {
			bucketMs := b * res60s
			minT := bucketMs + 1000
			maxT := bucketMs + 50000
			minV := float64(b % 20)
			maxV := float64(80 + (b % 20))
			if _, err := rollupStmt.Exec(tagID, res60s, bucketMs, minT, minV, maxT, maxV); err != nil {
				t.Fatalf("insert rollup: %v", err)
			}
		}
	}
	rollupStmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit fixture: %v", err)
	}

	// Measure 24h query with 12 tags and maxPoints = 2000
	start := time.Now()
	res, err := store.GetRange(tagIDs, 0, totalBuckets*res60s, 2000)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetRange 24h error: %v", err)
	}

	t.Logf("24h 12-tag GetRange completed in %v", elapsed)

	if elapsed >= time.Second {
		t.Fatalf("expected 24h 12-tag GetRange to complete in < 1s, took %v", elapsed)
	}

	for _, tid := range tagIDs {
		pts := res[tid]
		if len(pts) == 0 {
			t.Errorf("tag %s returned 0 points", tid)
		}
		if len(pts) > 2000 {
			t.Errorf("tag %s returned %d points, exceeding maxPoints=2000", tid, len(pts))
		}
	}
}
