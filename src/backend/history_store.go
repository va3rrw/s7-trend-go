package backend

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const historySchema = `
CREATE TABLE IF NOT EXISTS samples (
    tag_id TEXT NOT NULL,
    timestamp_ms INTEGER NOT NULL,
    value REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_samples_tag_time
    ON samples(tag_id, timestamp_ms);

CREATE TABLE IF NOT EXISTS samples_rollup (
    tag_id TEXT NOT NULL,
    res_ms INTEGER NOT NULL,
    bucket_ms INTEGER NOT NULL,
    min_t INTEGER NOT NULL,
    min_v REAL NOT NULL,
    max_t INTEGER NOT NULL,
    max_v REAL NOT NULL,
    PRIMARY KEY(tag_id, res_ms, bucket_ms)
) WITHOUT ROWID;
`

func defaultHistoryDatabasePath() string {
	return filepath.Join(getDefaultSettingsDir(), "history.db")
}

// RollupResolutions defines the pre-aggregated rollup bucket resolutions in milliseconds.
var RollupResolutions = []int64{1000, 10000, 60000}

// HistoryStore persists numeric samples in a local SQLite database with rollup tiers.
type HistoryStore struct {
	db           *sql.DB // writer DB, exported internally for tests
	writeDB      *sql.DB
	readDB       *sql.DB
	writeMu      sync.Mutex
	writeChan    chan storedSample
	flushChan    chan chan struct{}
	doneChan     chan struct{}
	wg           sync.WaitGroup
	droppedCount int64
}

// SamplePoint is the compact timestamp/value shape returned to the chart.
type SamplePoint struct {
	Timestamp int64   `json:"t"`
	Value     float64 `json:"v"`
}

type storedSample struct {
	TagID     string
	Timestamp int64
	Value     float64
}

type rollupRow struct {
	bucketMs int64
	minT     int64
	minV     float64
	maxT     int64
	maxV     float64
}

type bucketExtremes struct {
	minPoint SamplePoint
	maxPoint SamplePoint
	hasData  bool
}

func OpenHistoryStore(path string) (*HistoryStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}

	writeDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open history database writer: %w", err)
	}

	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)
	for _, statement := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		historySchema,
	} {
		if _, err := writeDB.Exec(statement); err != nil {
			writeDB.Close()
			return nil, fmt.Errorf("initialize history database: %w", err)
		}
	}

	readDB, err := sql.Open("sqlite", path)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("open history database reader pool: %w", err)
	}
	readDB.SetMaxOpenConns(4)
	readDB.SetMaxIdleConns(4)
	if _, err := readDB.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		writeDB.Close()
		readDB.Close()
		return nil, fmt.Errorf("set reader busy_timeout: %w", err)
	}

	// Backfill samples_rollup if samples exist but samples_rollup is empty
	var hasSamples, hasRollup bool
	_ = writeDB.QueryRow("SELECT EXISTS(SELECT 1 FROM samples LIMIT 1)").Scan(&hasSamples)
	_ = writeDB.QueryRow("SELECT EXISTS(SELECT 1 FROM samples_rollup LIMIT 1)").Scan(&hasRollup)
	if hasSamples && !hasRollup {
		for _, res := range RollupResolutions {
			_, _ = writeDB.Exec(`
INSERT INTO samples_rollup(tag_id, res_ms, bucket_ms, min_t, min_v, max_t, max_v)
SELECT
    tag_id,
    ? AS res_ms,
    (timestamp_ms / ?) * ? AS bucket_ms,
    timestamp_ms AS min_t,
    value AS min_v,
    timestamp_ms AS max_t,
    value AS max_v
FROM samples
ORDER BY tag_id, timestamp_ms
ON CONFLICT(tag_id, res_ms, bucket_ms) DO UPDATE SET
  min_t = CASE WHEN excluded.min_v < samples_rollup.min_v THEN excluded.min_t ELSE samples_rollup.min_t END,
  min_v = CASE WHEN excluded.min_v < samples_rollup.min_v THEN excluded.min_v ELSE samples_rollup.min_v END,
  max_t = CASE WHEN excluded.max_v > samples_rollup.max_v THEN excluded.max_t ELSE samples_rollup.max_t END,
  max_v = CASE WHEN excluded.max_v > samples_rollup.max_v THEN excluded.max_v ELSE samples_rollup.max_v END
`, res, res, res)
		}
	}

	store := &HistoryStore{
		db:        writeDB,
		writeDB:   writeDB,
		readDB:    readDB,
		writeChan: make(chan storedSample, 10000),
		flushChan: make(chan chan struct{}),
		doneChan:  make(chan struct{}),
	}
	store.wg.Add(1)
	go store.writerLoop()

	return store, nil
}

func (s *HistoryStore) writerLoop() {
	defer s.wg.Done()
	batch := make([]storedSample, 0, 1000)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.writeBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case sample := <-s.writeChan:
			batch = append(batch, sample)
			if len(batch) >= 1000 {
				flush()
			}
		case <-ticker.C:
			flush()
		case resp := <-s.flushChan:
			for {
				select {
				case sample := <-s.writeChan:
					batch = append(batch, sample)
				default:
					goto drained
				}
			}
		drained:
			flush()
			close(resp)
		case <-s.doneChan:
			for {
				select {
				case sample := <-s.writeChan:
					batch = append(batch, sample)
				default:
					goto doneDrained
				}
			}
		doneDrained:
			flush()
			return
		}
	}
}

func (s *HistoryStore) writeBatch(batch []storedSample) {
	if s == nil || s.writeDB == nil || len(batch) == 0 {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	insertStmt, err := tx.Prepare("INSERT INTO samples(tag_id, timestamp_ms, value) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	defer insertStmt.Close()

	upsertRollupStmt, err := tx.Prepare(`
INSERT INTO samples_rollup(tag_id, res_ms, bucket_ms, min_t, min_v, max_t, max_v)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(tag_id, res_ms, bucket_ms) DO UPDATE SET
  min_t = CASE WHEN excluded.min_v < samples_rollup.min_v THEN excluded.min_t ELSE samples_rollup.min_t END,
  min_v = CASE WHEN excluded.min_v < samples_rollup.min_v THEN excluded.min_v ELSE samples_rollup.min_v END,
  max_t = CASE WHEN excluded.max_v > samples_rollup.max_v THEN excluded.max_t ELSE samples_rollup.max_t END,
  max_v = CASE WHEN excluded.max_v > samples_rollup.max_v THEN excluded.max_v ELSE samples_rollup.max_v END
`)
	if err != nil {
		return
	}
	defer upsertRollupStmt.Close()

	for _, sample := range batch {
		if _, err := insertStmt.Exec(sample.TagID, sample.Timestamp, sample.Value); err != nil {
			return
		}
		for _, res := range RollupResolutions {
			bucket := (sample.Timestamp / res) * res
			if _, err := upsertRollupStmt.Exec(
				sample.TagID, res, bucket,
				sample.Timestamp, sample.Value,
				sample.Timestamp, sample.Value,
			); err != nil {
				return
			}
		}
	}

	_ = tx.Commit()
}

func (s *HistoryStore) Insert(sample storedSample) error {
	if s == nil || s.writeDB == nil {
		return fmt.Errorf("history database is not available")
	}

	select {
	case s.writeChan <- sample:
		return nil
	default:
		atomic.AddInt64(&s.droppedCount, 1)
		return nil
	}
}

// Flush blocks until all queued samples in the writer channel are written to the database.
func (s *HistoryStore) Flush() {
	if s == nil || s.flushChan == nil {
		return
	}
	resp := make(chan struct{})
	select {
	case s.flushChan <- resp:
		<-resp
	case <-s.doneChan:
	}
}

func (s *HistoryStore) GetRange(tagIDs []string, startMs, endMs int64, maxPoints int) (map[string][]SamplePoint, error) {
	result := make(map[string][]SamplePoint, len(tagIDs))
	if s == nil || s.readDB == nil {
		for _, tagID := range tagIDs {
			result[tagID] = []SamplePoint{}
		}
		return result, fmt.Errorf("history database is not available")
	}
	for _, tagID := range tagIDs {
		result[tagID] = []SamplePoint{}
	}
	if len(tagIDs) == 0 || startMs > endMs {
		return result, nil
	}

	// If maxPoints <= 0, return all raw points without downsampling
	if maxPoints <= 0 {
		return s.getRawRange(tagIDs, startMs, endMs, 0)
	}

	// Clamp maxPoints to [100, 20000]
	if maxPoints < 100 {
		maxPoints = 100
	} else if maxPoints > 20000 {
		maxPoints = 20000
	}

	numBuckets := maxPoints / 2
	span := endMs - startMs
	b := span / int64(numBuckets)

	// If bucket size b < 1000, query raw samples and downsample in Go if count > maxPoints
	if b < 1000 {
		return s.getRawRange(tagIDs, startMs, endMs, maxPoints)
	}

	// Pick coarsest rollup tier with res_ms <= b
	var resMs int64 = 1000
	if b >= 60000 {
		resMs = 60000
	} else if b >= 10000 {
		resMs = 10000
	}

	targetBucketSize := (span + int64(numBuckets) - 1) / int64(numBuckets)
	if targetBucketSize < 1 {
		targetBucketSize = 1
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
	args := make([]any, 0, len(tagIDs)+3)
	for _, tid := range tagIDs {
		args = append(args, tid)
	}
	startBucket := (startMs / resMs) * resMs
	endBucket := (endMs / resMs) * resMs
	args = append(args, resMs, startBucket, endBucket)

	rows, err := s.readDB.Query(
		"SELECT tag_id, bucket_ms, min_t, min_v, max_t, max_v FROM samples_rollup "+
			"WHERE tag_id IN ("+placeholders+") AND res_ms = ? AND bucket_ms >= ? AND bucket_ms <= ? "+
			"ORDER BY tag_id, bucket_ms",
		args...,
	)
	if err != nil {
		return s.getRawRange(tagIDs, startMs, endMs, maxPoints)
	}
	defer rows.Close()

	tagRollups := make(map[string][]rollupRow, len(tagIDs))
	totalRows := 0
	for rows.Next() {
		var tid string
		var r rollupRow
		if err := rows.Scan(&tid, &r.bucketMs, &r.minT, &r.minV, &r.maxT, &r.maxV); err != nil {
			return result, err
		}
		tagRollups[tid] = append(tagRollups[tid], r)
		totalRows++
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	// Fallback to raw query if rollup has no data
	if totalRows == 0 {
		return s.getRawRange(tagIDs, startMs, endMs, maxPoints)
	}

	for _, tid := range tagIDs {
		rollup := tagRollups[tid]
		if len(rollup) == 0 {
			result[tid] = []SamplePoint{}
			continue
		}
		result[tid] = mergeRollupBuckets(rollup, startMs, endMs, targetBucketSize, numBuckets)
	}

	return result, nil
}

func (s *HistoryStore) getRawRange(tagIDs []string, startMs, endMs int64, maxPoints int) (map[string][]SamplePoint, error) {
	result := make(map[string][]SamplePoint, len(tagIDs))
	for _, tid := range tagIDs {
		result[tid] = []SamplePoint{}
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
	args := make([]any, 0, len(tagIDs)+2)
	for _, tid := range tagIDs {
		args = append(args, tid)
	}
	args = append(args, startMs, endMs)

	rows, err := s.readDB.Query(
		"SELECT tag_id, timestamp_ms, value FROM samples "+
			"WHERE tag_id IN ("+placeholders+") AND timestamp_ms >= ? AND timestamp_ms <= ? "+
			"ORDER BY tag_id, timestamp_ms",
		args...,
	)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var tagID string
		var point SamplePoint
		if err := rows.Scan(&tagID, &point.Timestamp, &point.Value); err != nil {
			return result, err
		}
		result[tagID] = append(result[tagID], point)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	if maxPoints <= 0 {
		return result, nil
	}

	numBuckets := maxPoints / 2
	span := endMs - startMs
	targetBucketSize := (span + int64(numBuckets) - 1) / int64(numBuckets)
	if targetBucketSize < 1 {
		targetBucketSize = 1
	}

	for _, tid := range tagIDs {
		pts := result[tid]
		if len(pts) > maxPoints {
			result[tid] = downsamplePoints(pts, startMs, targetBucketSize, numBuckets)
		}
	}

	return result, nil
}

func mergeRollupBuckets(rows []rollupRow, startMs, endMs, targetBucketSize int64, numBuckets int) []SamplePoint {
	buckets := make([]bucketExtremes, numBuckets)

	addPoint := func(t int64, v float64) {
		if t < startMs || t > endMs {
			return
		}
		idx := int((t - startMs) / targetBucketSize)
		if idx < 0 {
			idx = 0
		} else if idx >= numBuckets {
			idx = numBuckets - 1
		}
		b := &buckets[idx]
		p := SamplePoint{Timestamp: t, Value: v}
		if !b.hasData {
			b.minPoint = p
			b.maxPoint = p
			b.hasData = true
		} else {
			if v < b.minPoint.Value {
				b.minPoint = p
			}
			if v > b.maxPoint.Value {
				b.maxPoint = p
			}
		}
	}

	for _, row := range rows {
		addPoint(row.minT, row.minV)
		if row.maxT != row.minT || row.maxV != row.minV {
			addPoint(row.maxT, row.maxV)
		}
	}

	result := make([]SamplePoint, 0, numBuckets*2)
	for _, b := range buckets {
		if !b.hasData {
			continue
		}
		if b.minPoint.Timestamp == b.maxPoint.Timestamp && b.minPoint.Value == b.maxPoint.Value {
			result = append(result, b.minPoint)
		} else if b.minPoint.Timestamp <= b.maxPoint.Timestamp {
			result = append(result, b.minPoint, b.maxPoint)
		} else {
			result = append(result, b.maxPoint, b.minPoint)
		}
	}
	return result
}

func downsamplePoints(points []SamplePoint, startMs, targetBucketSize int64, numBuckets int) []SamplePoint {
	if len(points) == 0 {
		return []SamplePoint{}
	}
	buckets := make([]bucketExtremes, numBuckets)

	for _, p := range points {
		idx := int((p.Timestamp - startMs) / targetBucketSize)
		if idx < 0 {
			idx = 0
		} else if idx >= numBuckets {
			idx = numBuckets - 1
		}
		b := &buckets[idx]
		if !b.hasData {
			b.minPoint = p
			b.maxPoint = p
			b.hasData = true
		} else {
			if p.Value < b.minPoint.Value {
				b.minPoint = p
			}
			if p.Value > b.maxPoint.Value {
				b.maxPoint = p
			}
		}
	}

	result := make([]SamplePoint, 0, numBuckets*2)
	for _, b := range buckets {
		if !b.hasData {
			continue
		}
		if b.minPoint.Timestamp == b.maxPoint.Timestamp && b.minPoint.Value == b.maxPoint.Value {
			result = append(result, b.minPoint)
		} else if b.minPoint.Timestamp <= b.maxPoint.Timestamp {
			result = append(result, b.minPoint, b.maxPoint)
		} else {
			result = append(result, b.maxPoint, b.minPoint)
		}
	}
	return result
}

func (s *HistoryStore) Latest(tagID string) (SamplePoint, bool, error) {
	if s == nil || s.readDB == nil {
		return SamplePoint{}, false, fmt.Errorf("history database is not available")
	}

	var point SamplePoint
	err := s.readDB.QueryRow(
		"SELECT timestamp_ms, value FROM samples WHERE tag_id = ? ORDER BY timestamp_ms DESC LIMIT 1",
		tagID,
	).Scan(&point.Timestamp, &point.Value)
	if err == sql.ErrNoRows {
		return SamplePoint{}, false, nil
	}
	if err != nil {
		return SamplePoint{}, false, err
	}
	return point, true, nil
}

func (s *HistoryStore) IterateAll(fn func(storedSample) error) error {
	if s == nil || s.readDB == nil {
		return fmt.Errorf("history database is not available")
	}

	rows, err := s.readDB.Query(
		"SELECT tag_id, timestamp_ms, value FROM samples ORDER BY timestamp_ms, tag_id",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sample storedSample
		if err := rows.Scan(&sample.TagID, &sample.Timestamp, &sample.Value); err != nil {
			return err
		}
		if err := fn(sample); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *HistoryStore) Clear() error {
	if s == nil || s.writeDB == nil {
		return fmt.Errorf("history database is not available")
	}

	s.Flush()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM samples"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM samples_rollup"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *HistoryStore) Close() error {
	if s == nil {
		return nil
	}

	select {
	case <-s.doneChan:
		return nil
	default:
		close(s.doneChan)
	}

	s.wg.Wait()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	var err1, err2 error
	if s.writeDB != nil {
		err1 = s.writeDB.Close()
	}
	if s.readDB != nil {
		err2 = s.readDB.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}
