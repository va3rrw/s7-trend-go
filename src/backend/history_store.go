package backend

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
`

func defaultHistoryDatabasePath() string {
	return filepath.Join(getDefaultSettingsDir(), "history.db")
}

// HistoryStore persists numeric samples in a local SQLite database.
// SQLite is intentionally kept behind this small API so the chart-facing
// application methods do not depend on SQL details.
type HistoryStore struct {
	db      *sql.DB
	writeMu sync.Mutex
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

func OpenHistoryStore(path string) (*HistoryStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open history database: %w", err)
	}

	// A single connection avoids SQLite connection-pool lock contention while
	// WAL lets chart reads proceed while the polling goroutine writes.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, statement := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		historySchema,
	} {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("initialize history database: %w", err)
		}
	}

	return &HistoryStore{db: db}, nil
}

func (s *HistoryStore) Insert(sample storedSample) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("history database is not available")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.Exec(
		"INSERT INTO samples(tag_id, timestamp_ms, value) VALUES (?, ?, ?)",
		sample.TagID,
		sample.Timestamp,
		sample.Value,
	)
	return err
}

func (s *HistoryStore) GetRange(tagIDs []string, startMs, endMs int64) (map[string][]SamplePoint, error) {
	result := make(map[string][]SamplePoint, len(tagIDs))
	if s == nil || s.db == nil {
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

	placeholders := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
	args := make([]any, 0, len(tagIDs)+2)
	for _, tagID := range tagIDs {
		args = append(args, tagID)
	}
	args = append(args, startMs, endMs)

	rows, err := s.db.Query(
		"SELECT tag_id, timestamp_ms, value FROM samples "+
			"WHERE tag_id IN ("+placeholders+") AND timestamp_ms >= ? AND timestamp_ms <= "+
			"? ORDER BY tag_id, timestamp_ms",
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
	return result, nil
}

func (s *HistoryStore) Latest(tagID string) (SamplePoint, bool, error) {
	if s == nil || s.db == nil {
		return SamplePoint{}, false, fmt.Errorf("history database is not available")
	}

	var point SamplePoint
	err := s.db.QueryRow(
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
	if s == nil || s.db == nil {
		return fmt.Errorf("history database is not available")
	}

	rows, err := s.db.Query(
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
	if s == nil || s.db == nil {
		return fmt.Errorf("history database is not available")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.Exec("DELETE FROM samples")
	return err
}

func (s *HistoryStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.db.Close()
}
