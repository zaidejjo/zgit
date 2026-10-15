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
	// Embedded single-writer workload: one connection serializes all
	// access at the driver level, so parallel dashboard overviews can
	// never hit SQLITE_BUSY. busy_timeout covers external writers
	// (a second zgit process on the same DB).
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		database.Close()
		return nil, fmt.Errorf("pragma wal: %w", err)
	}
	if _, err := database.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		database.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if _, err := database.Exec(`PRAGMA foreign_keys=ON;`); err != nil {
		database.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
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
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS ai_sessions(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		mode TEXT NOT NULL DEFAULT 'ask',
		created_at INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		return fmt.Errorf("migrate ai_sessions: %w", err)
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS ai_messages(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL REFERENCES ai_sessions(id) ON DELETE CASCADE,
		sort INTEGER NOT NULL DEFAULT 0,
		role TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		tool_call_id TEXT NOT NULL DEFAULT '',
		tool_calls TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		return fmt.Errorf("migrate ai_messages: %w", err)
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_ai_messages_session
		ON ai_messages(session_id, sort);`); err != nil {
		return fmt.Errorf("migrate ai_messages index: %w", err)
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

// SetProjectColor changes a project's accent color (empty clears it).
func (s *Store) SetProjectColor(id, color string) error {
	res, err := s.db.Exec(`UPDATE projects SET color = ? WHERE id = ?`, strings.TrimSpace(color), id)
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
	var out []models.Project
	for rows.Next() {
		var pr models.Project
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.Color); err != nil {
			continue
		}
		out = append(out, pr)
	}
	rows.Close()
	// Members are filled after the outer rows are closed: nesting a
	// second query inside the iteration deadlocks a single-connection
	// pool (and needlessly holds a read cursor).
	for i := range out {
		out[i].Repos = s.ProjectRepos(out[i].ID)
		if out[i].Repos == nil {
			out[i].Repos = []string{}
		}
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

// --- AI chat ---
//
// Chat history lives here instead of chats.json: one row per message,
// so saving a message is a single INSERT instead of rewriting every
// session on every turn.

// ChatSessionMeta is a session row plus its message count.
type ChatSessionMeta struct {
	ID           string
	Name         string
	Mode         string
	MessageCount int
	CreatedAt    int64
	UpdatedAt    int64
}

// ChatMessageRow is one stored message. ToolCallsJSON holds the
// marshaled tool calls ("" when none).
type ChatMessageRow struct {
	Role          string
	Content       string
	ToolCallID    string
	ToolCallsJSON string
	CreatedAt     int64
}

// UpsertChatSession creates or replaces a session row.
func (s *Store) UpsertChatSession(id, name, mode string, createdAt, updatedAt int64) error {
	_, err := s.db.Exec(`INSERT INTO ai_sessions(id, name, mode, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name, mode = excluded.mode,
			created_at = excluded.created_at, updated_at = excluded.updated_at`,
		id, name, mode, createdAt, updatedAt)
	return err
}

// TouchChatSession bumps a session's updated_at.
func (s *Store) TouchChatSession(id string, updatedAt int64) error {
	_, err := s.db.Exec(`UPDATE ai_sessions SET updated_at = ? WHERE id = ?`, updatedAt, id)
	return err
}

// CountChatSessions returns the number of stored sessions.
func (s *Store) CountChatSessions() int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM ai_sessions`).Scan(&n)
	return n
}

// ListChatSessions returns session metadata newest-first with message counts.
func (s *Store) ListChatSessions() []ChatSessionMeta {
	rows, err := s.db.Query(`SELECT s.id, s.name, s.mode, s.created_at, s.updated_at,
		(SELECT COUNT(*) FROM ai_messages m WHERE m.session_id = s.id)
		FROM ai_sessions s ORDER BY s.updated_at DESC`)
	if err != nil {
		return nil
	}
	var out []ChatSessionMeta
	for rows.Next() {
		var m ChatSessionMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.Mode, &m.CreatedAt, &m.UpdatedAt, &m.MessageCount); err != nil {
			continue
		}
		out = append(out, m)
	}
	rows.Close()
	if out == nil {
		out = []ChatSessionMeta{}
	}
	return out
}

// AppendChatMessage stores one message and bumps the session clock atomically.
func (s *Store) AppendChatMessage(sessionID, role, content, toolCallID, toolCallsJSON string, createdAt int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var sort int
	_ = tx.QueryRow(`SELECT COALESCE(MAX(sort), -1) + 1 FROM ai_messages WHERE session_id = ?`, sessionID).Scan(&sort)
	if _, err := tx.Exec(`INSERT INTO ai_messages(session_id, sort, role, content, tool_call_id, tool_calls, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		sessionID, sort, role, content, toolCallID, toolCallsJSON, createdAt); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE ai_sessions SET updated_at = ? WHERE id = ?`, createdAt, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

// GetChatMessages returns a session's messages in order.
func (s *Store) GetChatMessages(sessionID string) []ChatMessageRow {
	rows, err := s.db.Query(`SELECT role, content, tool_call_id, tool_calls, created_at
		FROM ai_messages WHERE session_id = ? ORDER BY sort ASC`, sessionID)
	if err != nil {
		return nil
	}
	var out []ChatMessageRow
	for rows.Next() {
		var m ChatMessageRow
		if err := rows.Scan(&m.Role, &m.Content, &m.ToolCallID, &m.ToolCallsJSON, &m.CreatedAt); err != nil {
			continue
		}
		out = append(out, m)
	}
	rows.Close()
	if out == nil {
		out = []ChatMessageRow{}
	}
	return out
}

// ReplaceChatMessages swaps a session's messages wholesale (used by clear).
func (s *Store) ReplaceChatMessages(sessionID string, msgs []ChatMessageRow) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ai_messages WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	for i, m := range msgs {
		if _, err := tx.Exec(`INSERT INTO ai_messages(session_id, sort, role, content, tool_call_id, tool_calls, created_at)
			VALUES(?, ?, ?, ?, ?, ?, ?)`,
			sessionID, i, m.Role, m.Content, m.ToolCallID, m.ToolCallsJSON, m.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RenameChatSession changes a session's name.
func (s *Store) RenameChatSession(id, name string, updatedAt int64) error {
	res, err := s.db.Exec(`UPDATE ai_sessions SET name = ?, updated_at = ? WHERE id = ?`, name, updatedAt, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("chat session not found: %s", id)
	}
	return nil
}

// DeleteChatSession removes a session and its messages.
func (s *Store) DeleteChatSession(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ai_messages WHERE session_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM ai_sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("chat session not found: %s", id)
	}
	return tx.Commit()
}
