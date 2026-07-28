package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaidejjo/zgit/pkg/core/ai"
	"github.com/zaidejjo/zgit/pkg/core/config"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// Stream view IDs (negative to avoid overlap with real views 0-4).
const (
	aiStreamViewID = -2
)

// streamTokenMsg is sent per token during streaming.
type streamTokenMsg struct {
	token string
}

// streamDoneMsg signals streaming completion.
type streamDoneMsg struct {
	content string
	err     error
}

// AI state on the Model.
type aiState struct {
	Provider string
	Model    string
	APIKey   string
	Endpoint string
	Agent    *ai.Agent
	Session  *ai.SessionManager
	Sidebar  views.AISidebarModel
}

// initAI loads config and initializes AI state. Safe to call even without AI setup.
func (m *Model) initAI() {
	cfg, err := config.New()
	if err != nil {
		return
	}

	m.aiData.Provider = cfg.GetString("ai.provider")
	m.aiData.APIKey = cfg.GetString("ai.api_key")
	m.aiData.Model = cfg.GetString("ai.model")
	m.aiData.Endpoint = cfg.GetString("ai.endpoint")

	// Init unified sidebar
	m.aiData.Sidebar = views.NewAISidebarModel()
	m.aiData.Sidebar.Provider = m.aiData.Provider
	m.aiData.Sidebar.Model = m.aiData.Model

	// Session manager
	m.aiData.Session = ai.NewSessionManager()

	if m.aiData.Provider != "" && m.aiData.APIKey != "" {
		_, _ = m.aiData.Session.Create("Ask", "ask")
	}
}

// aiProviderLabel returns a short string like "openai:gpt-4o" for the status bar.
func (m *Model) aiProviderLabel() string {
	if m.aiData.Provider == "" {
		return ""
	}
	label := m.aiData.Provider
	if m.aiData.Model != "" {
		label += ":" + m.aiData.Model
	}
	return label
}

// handleAIKey routes key events to the sidebar and triggers send on enter.
func (m *Model) handleAIKey(key string) (tea.Model, tea.Cmd) {
	// Toggle keys close sidebar
	if key == "ctrl+g" || key == "ctrl+e" {
		m.aiData.Sidebar.Close()
		m.calcLayout()
		return m, nil
	}

	wasActive := m.aiData.Sidebar.Active()
	handled := m.aiData.Sidebar.HandleKey(key)
	if !handled {
		return m, nil
	}

	// Recalculate layout if sidebar state changed (open/close)
	if wasActive != m.aiData.Sidebar.Active() {
		m.calcLayout()
	}

	// On enter in input state, start streaming
	if key == "enter" && m.aiData.Sidebar.State == views.SBSInput {
		input := m.aiData.Sidebar.GetInput()
		if input != "" {
			return m, m.startSidebarStream(input)
		}
	}

	return m, nil
}

// startSidebarStream begins a streaming AI response.
func (m *Model) startSidebarStream(input string) tea.Cmd {
	m.aiData.Sidebar.AddMessage("user", input)
	m.aiData.Sidebar.Input = ""
	m.aiData.Sidebar.Cursor = 0
	m.aiData.Sidebar.State = views.SBSStreaming
	m.aiData.Sidebar.StreamingContent = ""
	m.aiData.Sidebar.Error = ""

	if m.aiData.Provider == "" || m.aiData.APIKey == "" {
		m.aiData.Sidebar.Error = "AI not configured. Set up in config."
		m.aiData.Sidebar.State = views.SBSInput
		return nil
	}

	// Build context-aware messages
	aiCfg := ai.Config{
		Provider: ai.ProviderKind(m.aiData.Provider),
		APIKey:   m.aiData.APIKey,
		Model:    m.aiData.Model,
		Endpoint: m.aiData.Endpoint,
	}

	messages := buildAIContextMessages(input, m.status.Status)

	// Start streaming in goroutine — tokens sent through msgs channel (non-blocking)
	go func() {
		provider, err := ai.NewAskProvider(aiCfg)
		if err != nil {
			m.sendStreamDone("", fmt.Errorf("create provider: %w", err))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60e9)
		defer cancel()

		var buf strings.Builder
		onToken := func(token string) {
			buf.WriteString(token)
			// Non-blocking send — if channel full, skip token to avoid deadlock
			select {
			case m.msgs <- teaMsg{view: aiStreamViewID, data: streamTokenMsg{token: token}}:
			default:
			}
		}

		resp, err := provider.AskStream(ctx, messages, onToken)
		if err != nil {
			m.sendStreamDone(buf.String(), fmt.Errorf("stream error: %w", err))
			return
		}
		if resp.Content != "" {
			buf.WriteString(resp.Content)
		}
		m.sendStreamDone(buf.String(), nil)
	}()

	return nil
}

// buildAIContextMessages creates the message list with repo context.
func buildAIContextMessages(input string, status *models.Status) []ai.Message {
	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful Git assistant. Answer questions about the repository, explain Git concepts, and help the user understand their code. Be concise and accurate."},
	}

	// Add repo context if available
	if status != nil {
		contextInfo := fmt.Sprintf("Current branch: %s\n", status.Branch)
		if status.Upstream != "" {
			contextInfo += fmt.Sprintf("Upstream: %s (ahead: %d, behind: %d)\n",
				status.Upstream, status.Ahead, status.Behind)
		}
		messages = append(messages, ai.Message{Role: "system", Content: contextInfo})
	}

	messages = append(messages, ai.Message{Role: "user", Content: input})
	return messages
}

// sendStreamDone sends a stream completion message (non-blocking).
func (m *Model) sendStreamDone(content string, err error) {
	select {
	case m.msgs <- teaMsg{view: aiStreamViewID, data: streamDoneMsg{content: content, err: err}}:
	default:
	}
}

// handleAIStreamMsg processes streaming token and done messages.
func (m *Model) handleAIStreamMsg(data interface{}) {
	switch d := data.(type) {
	case streamTokenMsg:
		m.aiData.Sidebar.StreamingContent += d.token
	case streamDoneMsg:
		if d.err != nil {
			m.aiData.Sidebar.Error = d.err.Error()
		} else if d.content != "" {
			m.aiData.Sidebar.AddMessage("assistant", d.content)
		}
		m.aiData.Sidebar.StreamingContent = ""
		m.aiData.Sidebar.State = views.SBSInput
	}
}
