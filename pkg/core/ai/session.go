package ai

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
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
// Thread-safe. Sessions are stored in memory only (no disk persistence).
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	activeID string
	nextID   atomic.Int64
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
	s.Name = name
	s.UpdatedAt = time.Now()
	return nil
}

// Delete removes a session by ID. If it was active, activeID is cleared.
func (sm *SessionManager) Delete(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessions[id]; !ok {
		return fmt.Errorf("session %q not found", id)
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
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
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
