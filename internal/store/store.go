package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "modernc.org/sqlite"

	"github.com/hd-health/hd-health/internal/domain"
)

type Store struct {
	db *sql.DB
}

func DefaultDBPath() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "hd-health", "metrics.db")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "hd-health", "metrics.db")
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultDBPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS volume_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  collected_at TEXT NOT NULL,
  mount TEXT NOT NULL,
  name TEXT,
  capacity_bytes INTEGER NOT NULL,
  free_bytes INTEGER NOT NULL,
  used_bytes INTEGER NOT NULL,
  used_percent REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_vol_mount_time ON volume_snapshots(mount, collected_at);
`)
	return err
}

func (s *Store) RecordVolumes(vols []domain.Volume) error {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO volume_snapshots (collected_at, mount, name, capacity_bytes, free_bytes, used_bytes, used_percent) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, v := range vols {
		if _, err := stmt.Exec(now, v.Mount, v.Name, v.Capacity, v.Free, v.Used, v.UsedPercent); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type HistoryPoint struct {
	CollectedAt time.Time
	UsedBytes   int64
	FreeBytes   int64
	Capacity    int64
}

func (s *Store) VolumeHistory(mount string, since time.Time) ([]HistoryPoint, error) {
	rows, err := s.db.Query(`
SELECT collected_at, used_bytes, free_bytes, capacity_bytes FROM volume_snapshots
WHERE mount = ? AND collected_at >= ? ORDER BY collected_at ASC`,
		mount, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pts []HistoryPoint
	for rows.Next() {
		var ts string
		var used, free, cap int64
		if err := rows.Scan(&ts, &used, &free, &cap); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339, ts)
		pts = append(pts, HistoryPoint{CollectedAt: t, UsedBytes: used, FreeBytes: free, Capacity: cap})
	}
	return pts, rows.Err()
}

func (s *Store) GrowthBytesPerDay(mount string, days int) (float64, error) {
	if days < 1 {
		days = 7
	}
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	pts, err := s.VolumeHistory(mount, since)
	if err != nil {
		return 0, err
	}
	if len(pts) < 2 {
		return 0, nil
	}
	first, last := pts[0], pts[len(pts)-1]
	dt := last.CollectedAt.Sub(first.CollectedAt).Hours() / 24
	if dt < 0.01 {
		return 0, nil
	}
	growth := float64(last.UsedBytes-first.UsedBytes) / dt
	return growth, nil
}

func (s *Store) DBPath() string {
	var path string
	_ = s.db.QueryRow("PRAGMA database_list").Scan(nil, &path, nil)
	return path
}

func (s *Store) String() string {
	return fmt.Sprintf("sqlite:%s", s.DBPath())
}
