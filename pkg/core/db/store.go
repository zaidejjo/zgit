// Package db is the local SQLite store for repository bookkeeping
// (recents, favorites, projects). Pure-Go driver (modernc.org/sqlite):
// no CGO, so Windows/darwin cross-compiles keep working.
package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/models"
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
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS overview_cache(
		path TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		fetched_at INTEGER NOT NULL
	);`); err != nil {
		return fmt.Errorf("migrate overview_cache: %w", err)
	}
	return nil
}

// CachedOverviewTTLSeconds bounds how long a snapshot is served as-is.
const CachedOverviewTTLSeconds = 120

// GetCachedOverview returns the snapshot when fresher than ttlSeconds.
func (s *Store) GetCachedOverview(path string, ttlSeconds int64) (*models.RepoOverview, bool) {
	var data string
	var fetchedAt int64
	if err := s.db.QueryRow(`SELECT data, fetched_at FROM overview_cache WHERE path = ?`, path).Scan(&data, &fetchedAt); err != nil {
		return nil, false
	}
	if time.Now().Unix()-fetchedAt > ttlSeconds {
		return nil, false
	}
	var ov models.RepoOverview
	if err := json.Unmarshal([]byte(data), &ov); err != nil {
		return nil, false
	}
	return &ov, true
}

// SetCachedOverview stores a snapshot (best effort by callers).
func (s *Store) SetCachedOverview(ov *models.RepoOverview) error {
	data, err := json.Marshal(ov)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO overview_cache(path, data, fetched_at) VALUES(?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET data = excluded.data, fetched_at = excluded.fetched_at`,
		ov.Path, string(data), time.Now().Unix())
	return err
}

// InvalidateOverview drops one cached snapshot.
func (s *Store) InvalidateOverview(path string) {
	_, _ = s.db.Exec(`DELETE FROM overview_cache WHERE path = ?`, path)
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
// its project memberships and cached overview.
func (s *Store) RemoveRepo(path string) error {
	if _, err := s.db.Exec(`DELETE FROM project_repos WHERE path = ?`, path); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM overview_cache WHERE path = ?`, path); err != nil {
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
		if _, err := s.db.Exec(`DELETE FROM overview_cache WHERE path = ?`, p); err != nil {
			return 0, err
		}
	}
	return len(dead), nil
}

// --- Projects ---

// CreateProject creates a project and returns its id.
func (s *Store) CreateProject(name, color string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("project name is required")
	}
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate project id: %w", err)
	}
	id := hex.EncodeToString(b[:])
	var sort int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort), -1) + 1 FROM projects`).Scan(&sort)
	if _, err := s.db.Exec(`INSERT INTO projects(id, name, color, sort) VALUES(?, ?, ?, ?)`,
		id, name, strings.TrimSpace(color), sort); err != nil {
		return "", err
	}
	return id, nil
}

// RenameProject changes a project's name.
func (s *Store) RenameProject(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	res, err := s.db.Exec(`UPDATE projects SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

// DeleteProject removes a project and its memberships (repos stay).
func (s *Store) DeleteProject(id string) error {
	if _, err := s.db.Exec(`DELETE FROM project_repos WHERE project_id = ?`, id); err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

// ProjectRepos returns member paths in sort order.
func (s *Store) ProjectRepos(id string) []string {
	rows, err := s.db.Query(`SELECT path FROM project_repos WHERE project_id = ? ORDER BY sort ASC`, id)
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

// ListProjects returns all projects with members, in sort order.
func (s *Store) ListProjects() []models.Project {
	rows, err := s.db.Query(`SELECT id, name, color FROM projects ORDER BY sort ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.Project
	for rows.Next() {
		var pr models.Project
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.Color); err != nil {
			continue
		}
		pr.Repos = s.ProjectRepos(pr.ID)
		if pr.Repos == nil {
			pr.Repos = []string{}
		}
		out = append(out, pr)
	}
	if out == nil {
		out = []models.Project{}
	}
	return out
}

// AddProjectRepo appends a repo to a project (idempotent).
func (s *Store) AddProjectRepo(id, path string) error {
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = ?`, id).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	var sort int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort), -1) + 1 FROM project_repos WHERE project_id = ?`, id).Scan(&sort)
	_, err := s.db.Exec(`INSERT INTO project_repos(project_id, path, sort) VALUES(?, ?, ?)
		ON CONFLICT(project_id, path) DO NOTHING`, id, path, sort)
	return err
}

// RemoveProjectRepo drops a repo from a project (repo row stays).
func (s *Store) RemoveProjectRepo(id, path string) error {
	_, err := s.db.Exec(`DELETE FROM project_repos WHERE project_id = ? AND path = ?`, id, path)
	return err
}

// SetProjectRepos replaces a project's membership order wholesale
// (used by drag-and-drop reorder). Unknown paths are ignored.
func (s *Store) SetProjectRepos(id string, paths []string) error {
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = ?`, id).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM project_repos WHERE project_id = ?`, id); err != nil {
		return err
	}
	seen := make(map[string]bool)
	sort := 0
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if _, err := tx.Exec(`INSERT INTO project_repos(project_id, path, sort) VALUES(?, ?, ?)`, id, p, sort); err != nil {
			return err
		}
		sort++
	}
	return tx.Commit()
}
