// Package db is the local SQLite store for repository bookkeeping
// (recents, favorites, projects). Pure-Go driver (modernc.org/sqlite):
// no CGO, so Windows/darwin cross-compiles keep working.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schemaVersion = 1

// Store wraps a SQLite database holding app-local state.
type Store struct {
	db *sql.DB
}

// Open creates the parent dir if needed, opens (or creates) the database
// at path, and runs migrations.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	// Single-writer workload; WAL avoids reader/writer lock contention
	// between the UI thread and background refreshes.
	if _, err := database.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		database.Close()
		return nil, fmt.Errorf("pragma wal: %w", err)
	}
	s := &Store{db: database}
	if err := s.migrate(); err != nil {
		database.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS meta(k TEXT PRIMARY KEY, v TEXT);`); err != nil {
		return fmt.Errorf("migrate meta: %w", err)
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS repos(
		path TEXT PRIMARY KEY,
		last_opened INTEGER NOT NULL DEFAULT 0,
		favorite INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		return fmt.Errorf("migrate repos: %w", err)
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS projects(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		color TEXT NOT NULL DEFAULT '',
		sort INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		return fmt.Errorf("migrate projects: %w", err)
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS project_repos(
		project_id TEXT NOT NULL,
		path TEXT NOT NULL,
		sort INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY(project_id, path)
	);`); err != nil {
		return fmt.Errorf("migrate project_repos: %w", err)
	}
	if _, err := s.db.Exec(`INSERT OR IGNORE INTO meta(k, v) VALUES('schema_version', ?)`, schemaVersion); err != nil {
		return fmt.Errorf("migrate version: %w", err)
	}
	return nil
}

func (s *Store) getMeta(key string) string {
	var v string
	_ = s.db.QueryRow(`SELECT v FROM meta WHERE k = ?`, key).Scan(&v)
	return v
}

func (s *Store) setMeta(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO meta(k, v) VALUES(?, ?)
		ON CONFLICT(k) DO UPDATE SET v = excluded.v`, key, value)
	return err
}

// Migrated reports whether the one-time YAML import already ran.
func (s *Store) Migrated() bool {
	return s.getMeta("yaml_imported") == "1"
}

// MarkMigrated records that the YAML import ran.
func (s *Store) MarkMigrated() error {
	return s.setMeta("yaml_imported", "1")
}

// ImportRecents seeds last_opened timestamps, oldest first.
func (s *Store) ImportRecents(paths []string) error {
	now := time.Now().Unix()
	for i := len(paths) - 1; i >= 0; i-- {
		if _, err := s.db.Exec(`INSERT INTO repos(path, last_opened)
			VALUES(?, ?) ON CONFLICT(path) DO NOTHING`, paths[i], now-int64(len(paths)-i)); err != nil {
			return err
		}
	}
	return nil
}

// ImportFavorites marks paths as favorite (creating rows as needed).
func (s *Store) ImportFavorites(paths []string) error {
	for _, p := range paths {
		if _, err := s.db.Exec(`INSERT INTO repos(path, favorite) VALUES(?, 1)
			ON CONFLICT(path) DO UPDATE SET favorite = 1`, p); err != nil {
			return err
		}
	}
	return nil
}

// TrackRepo records an open: bumps last_opened, preserves favorite flag.
func (s *Store) TrackRepo(path string) error {
	_, err := s.db.Exec(`INSERT INTO repos(path, last_opened) VALUES(?, ?)
		ON CONFLICT(path) DO UPDATE SET last_opened = excluded.last_opened`,
		path, time.Now().Unix())
	return err
}

// RemoveRepo forgets a repo (files on disk are untouched). Also drops
// its project memberships.
func (s *Store) RemoveRepo(path string) error {
	if _, err := s.db.Exec(`DELETE FROM project_repos WHERE path = ?`, path); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM repos WHERE path = ?`, path)
	return err
}

// SetFavorite pins (true) or unpins (false) a repo, creating its row.
func (s *Store) SetFavorite(path string, fav bool) error {
	v := 0
	if fav {
		v = 1
	}
	_, err := s.db.Exec(`INSERT INTO repos(path, favorite) VALUES(?, ?)
		ON CONFLICT(path) DO UPDATE SET favorite = excluded.favorite`, path, v)
	return err
}

// IsFavorite reports whether path is pinned.
func (s *Store) IsFavorite(path string) bool {
	var v int
	if err := s.db.QueryRow(`SELECT favorite FROM repos WHERE path = ?`, path).Scan(&v); err != nil {
		return false
	}
	return v == 1
}

// DashboardRepos returns favorites (by name) then recents (newest first).
func (s *Store) DashboardRepos() []string {
	rows, err := s.db.Query(`SELECT path FROM repos
		ORDER BY favorite DESC, CASE WHEN favorite = 1 THEN path ELSE '' END ASC, last_opened DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

// PruneDead removes rows whose paths no longer exist on disk.
// Favorites are kept (drives unmount). Returns rows removed.
func (s *Store) PruneDead(exists func(path string) bool) (int, error) {
	rows, err := s.db.Query(`SELECT path FROM repos WHERE favorite = 0`)
	if err != nil {
		return 0, err
	}
	var dead []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		if !exists(p) {
			dead = append(dead, p)
		}
	}
	rows.Close()
	for _, p := range dead {
		if _, err := s.db.Exec(`DELETE FROM repos WHERE path = ?`, p); err != nil {
			return 0, err
		}
	}
	return len(dead), nil
}
