package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/db"
)

// Session represents a single AI chat session (Ask or Agent mode).
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mode      string    `json:"mode"` // "ask" | "agent"
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionSummary is a lightweight view of a session (no messages).
type SessionSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Mode         string    `json:"mode"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

const (
	MaxSessions      = 50
	DefaultAskPrompt = "You are a helpful Git assistant. Answer questions about the repository, explain Git concepts, and help the user understand their code. Be concise and accurate."
)

// SessionManager manages in-memory AI chat sessions.
// Thread-safe. When a Store is attached (AttachStore), every mutation
// is written through to SQLite, so chat history survives restarts
// without rewriting a whole JSON file per message.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	activeID string
	nextID   atomic.Int64
	store    *db.Store
}

// AttachStore enables SQLite persistence for all future mutations.
func (sm *SessionManager) AttachStore(store *db.Store) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.store = store
}

// LoadFromStore replaces memory with every stored session (newest
// activity first for nextID tracking only; List() sorts on read).
func (sm *SessionManager) LoadFromStore() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.store == nil {
		return fmt.Errorf("no store attached")
	}
	metas := sm.store.ListChatSessions()
	fresh := make(map[string]*Session, len(metas))
	for _, m := range metas {
		rows := sm.store.GetChatMessages(m.ID)
		msgs := make([]Message, 0, len(rows))
		for _, r := range rows {
			msg := Message{Role: r.Role, Content: r.Content, ToolCallID: r.ToolCallID}
			if r.ToolCallsJSON != "" {
				var calls []ToolCall
				if err := json.Unmarshal([]byte(r.ToolCallsJSON), &calls); err != nil {
					return fmt.Errorf("decode tool calls for %s: %w", m.ID, err)
				}
				msg.ToolCalls = calls
			}
			msgs = append(msgs, msg)
		}
		fresh[m.ID] = &Session{
			ID:        m.ID,
			Name:      m.Name,
			Mode:      m.Mode,
			Messages:  msgs,
			CreatedAt: time.Unix(m.CreatedAt, 0),
			UpdatedAt: time.Unix(m.UpdatedAt, 0),
		}
		sm.trackMaxID(m.ID)
	}
	sm.sessions = fresh
	sm.activeID = ""
	return nil
}

// MigrateFileToStore imports a legacy chats.json once: when the store
// holds no sessions yet, the file is loaded into memory and persisted.
// Returns true when a migration ran.
func (sm *SessionManager) MigrateFileToStore(path string) (bool, error) {
	if sm.store == nil {
		return false, fmt.Errorf("no store attached")
	}
	if sm.store.CountChatSessions() > 0 {
		return false, nil
	}
	if _, err := os.Stat(path); err != nil {
		return false, nil // nothing to migrate
	}
	if err := sm.LoadFromFile(path); err != nil {
		return false, err
	}
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, s := range sm.sessions {
		if err := sm.persistSession(s); err != nil {
			return false, err
		}
	}
	return true, nil
}

// trackMaxID keeps numeric session IDs collision-free. Callers hold sm.mu.
func (sm *SessionManager) trackMaxID(id string) {
	var n int64
	if _, err := fmt.Sscanf(id, "session_%d", &n); err == nil && n > sm.nextID.Load() {
		sm.nextID.Store(n)
	}
}

// persistSession upserts one in-memory session with all its messages.
// Callers hold sm.mu.
func (sm *SessionManager) persistSession(s *Session) error {
	if sm.store == nil {
		return nil
	}
	if err := sm.store.UpsertChatSession(s.ID, s.Name, s.Mode,
		s.CreatedAt.Unix(), s.UpdatedAt.Unix()); err != nil {
		return err
	}
	rows := make([]db.ChatMessageRow, 0, len(s.Messages))
	for _, m := range s.Messages {
		var tcJSON string
		if len(m.ToolCalls) > 0 {
			data, err := json.Marshal(m.ToolCalls)
			if err != nil {
				return fmt.Errorf("encode tool calls: %w", err)
			}
			tcJSON = string(data)
		}
		rows = append(rows, db.ChatMessageRow{
			Role:          m.Role,
			Content:       m.Content,
			ToolCallID:    m.ToolCallID,
			ToolCallsJSON: tcJSON,
			CreatedAt:     s.CreatedAt.Unix(),
		})
	}
	return sm.store.ReplaceChatMessages(s.ID, rows)
}

// NewSessionManager creates a new in-memory session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session with the given name and mode.
// Initializes with the appropriate system message.
func (sm *SessionManager) Create(name, mode string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.sessions) >= MaxSessions {
		return nil, fmt.Errorf("max sessions reached (%d)", MaxSessions)
	}

	id := fmt.Sprintf("session_%d", sm.nextID.Add(1))
	now := time.Now()

	sysContent := DefaultAskPrompt
	sysRole := "system"

	session := &Session{
		ID:        id,
		Name:      name,
		Mode:      mode,
		Messages:  []Message{{Role: sysRole, Content: sysContent}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	sm.sessions[id] = session
	sm.activeID = id
	if err := sm.persistSession(session); err != nil {
		delete(sm.sessions, id)
		return nil, err
	}
	return session, nil
}

// List returns summaries of all sessions, sorted by UpdatedAt descending.
func (sm *SessionManager) List() []SessionSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summaries := make([]SessionSummary, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		summaries = append(summaries, SessionSummary{
			ID:           s.ID,
			Name:         s.Name,
			Mode:         s.Mode,
			MessageCount: len(s.Messages),
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
		})
	}

	// Sort by UpdatedAt desc (simple insertion-style for small N)
	for i := 0; i < len(summaries); i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].UpdatedAt.After(summaries[i].UpdatedAt) {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}

	return summaries
}

// Get returns a session by ID. Returns nil if not found.
func (sm *SessionManager) Get(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

// Active returns the currently active session. Returns nil if none.
func (sm *SessionManager) Active() *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.activeID == "" {
		return nil
	}
	return sm.sessions[sm.activeID]
}

// ActiveID returns the ID of the active session.
func (sm *SessionManager) ActiveID() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.activeID
}

// Switch sets the active session to the given ID.
func (sm *SessionManager) Switch(id string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	sm.activeID = id
	return s, nil
}

// Rename updates the name of a session.
func (sm *SessionManager) Rename(id, name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	now := time.Now()
	s.Name = name
	s.UpdatedAt = now
	if sm.store != nil {
		if err := sm.store.RenameChatSession(id, name, now.Unix()); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a session by ID. If it was active, activeID is cleared.
func (sm *SessionManager) Delete(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessions[id]; !ok {
		return fmt.Errorf("session %q not found", id)
	}
	if sm.store != nil {
		if err := sm.store.DeleteChatSession(id); err != nil {
			return err
		}
	}
	delete(sm.sessions, id)
	if sm.activeID == id {
		sm.activeID = ""
	}
	return nil
}

// AddMessage appends a message to the session and updates UpdatedAt.
func (sm *SessionManager) AddMessage(id string, msg Message) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	now := time.Now()
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = now
	if sm.store != nil {
		var tcJSON string
		if len(msg.ToolCalls) > 0 {
			data, err := json.Marshal(msg.ToolCalls)
			if err != nil {
				return fmt.Errorf("encode tool calls: %w", err)
			}
			tcJSON = string(data)
		}
		if err := sm.store.AppendChatMessage(id, msg.Role, msg.Content,
			msg.ToolCallID, tcJSON, now.Unix()); err != nil {
			// Roll back the in-memory append so both stay in sync.
			s.Messages = s.Messages[:len(s.Messages)-1]
			return err
		}
	}
	return nil
}

// GetMessages returns a copy of the session's messages.
func (sm *SessionManager) GetMessages(id string) ([]Message, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	out := make([]Message, len(s.Messages))
	copy(out, s.Messages)
	return out, nil
}

// ClearMessages clears all messages in a session, keeping only the system prompt.
func (sm *SessionManager) ClearMessages(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	// Keep only system messages
	var systemMsgs []Message
	for _, m := range s.Messages {
		if m.Role == "system" {
			systemMsgs = append(systemMsgs, m)
		}
	}
	if len(systemMsgs) == 0 {
		systemMsgs = []Message{{Role: "system", Content: DefaultAskPrompt}}
	}
	s.Messages = systemMsgs
	s.UpdatedAt = time.Now()
	if sm.store != nil {
		rows := make([]db.ChatMessageRow, 0, len(systemMsgs))
		for _, m := range systemMsgs {
			rows = append(rows, db.ChatMessageRow{Role: m.Role, Content: m.Content})
		}
		if err := sm.store.ReplaceChatMessages(id, rows); err != nil {
			return err
		}
	}
	return nil
}

// SaveToFile persists all sessions as JSON to the specified path.
func (sm *SessionManager) SaveToFile(path string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create chat store dir: %w", err)
	}

	data, err := json.MarshalIndent(sm.sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write chat store: %w", err)
	}
	return nil
}

// LoadFromFile loads sessions from a JSON file. Existing sessions are merged
// (file sessions overwrite in-memory ones with the same ID).
func (sm *SessionManager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // not an error
		}
		return fmt.Errorf("read chat store: %w", err)
	}

	var fileSessions map[string]*Session
	if err := json.Unmarshal(data, &fileSessions); err != nil {
		return fmt.Errorf("unmarshal sessions: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Track highest ID so we don't collide
	for id := range fileSessions {
		if _, exists := sm.sessions[id]; !exists {
			sm.sessions[id] = fileSessions[id]
		}
		sm.trackMaxID(id)
	}

	// Set active to most recent session if none active
	if sm.activeID == "" {
		var newest *Session
		for _, s := range fileSessions {
			if newest == nil || s.UpdatedAt.After(newest.UpdatedAt) {
				newest = s
			}
		}
		if newest != nil {
			sm.activeID = newest.ID
		}
	}

	return nil
}

// SetMode changes the mode of a session.
func (sm *SessionManager) SetMode(id, mode string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	s.Mode = mode
	s.UpdatedAt = time.Now()
	return nil
}
