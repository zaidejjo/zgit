package views

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// AISidebarMode selects ask vs agent mode.
type AISidebarMode int

const (
	ModeAsk AISidebarMode = iota
	ModeAgent
)

// AISidebarState tracks the sidebar UI phase.
type AISidebarState int

const (
	SBSClosed    AISidebarState = iota
	SBSInput                    // awaiting user input
	SBSStreaming                // token streaming in progress
	SBSProposals                // agent proposals shown (agent mode only)
	SBSModels                   // /models picker
	SBSChats                    // /chats session list
)

// ChatMessage is one turn in the conversation.
type ChatMessage struct {
	Role    string // "user", "assistant", "system"
	Content string
}

// AISidebarModel is the unified AI slide-over sidebar.
type AISidebarModel struct {
	Mode  AISidebarMode
	State AISidebarState

	Width  int
	Height int

	// Input
	Input  string
	Cursor int

	// Conversation history
	Messages []ChatMessage

	// Streaming content (accumulated while streaming)
	StreamingContent string

	// Agent proposals (type defined in ai_agent.go)
	Proposals     []AgentProposalCard
	ProposalFocus int

	// Sessions
	Sessions  []string
	SessionID string

	// Metadata
	Provider string
	Model    string

	// Error
	Error string
}

// AgentProposalCard represents a proposed git action from the agent.
type AgentProposalCard struct {
	ID          string
	Description string
	Status      string
	CanApprove  bool
	CanReject   bool
}

// NewAISidebarModel creates a default sidebar model.
func NewAISidebarModel() AISidebarModel {
	return AISidebarModel{
		State: SBSClosed,
		Mode:  ModeAsk,
	}
}

// Open opens the sidebar in the given mode.
func (m *AISidebarModel) Open(mode AISidebarMode) {
	m.Mode = mode
	m.State = SBSInput
	m.Input = ""
	m.Cursor = 0
	m.Error = ""
	m.StreamingContent = ""
}

// Close dismisses the sidebar.
func (m *AISidebarModel) Close() {
	m.State = SBSClosed
	m.StreamingContent = ""
	m.Error = ""

}

// Active returns true if sidebar is visible.
func (m *AISidebarModel) Active() bool {
	return m.State != SBSClosed
}

// GetInput returns trimmed input text.
func (m *AISidebarModel) GetInput() string {
	return strings.TrimSpace(m.Input)
}

// AddMessage appends a chat message.
func (m *AISidebarModel) AddMessage(role, content string) {
	m.Messages = append(m.Messages, ChatMessage{Role: role, Content: content})
}

// ClearMessages resets conversation.
func (m *AISidebarModel) ClearMessages() {
	m.Messages = nil
}

// --- Slash command detection ---

var slashActive bool

// SlashCmdActive returns true if input starts with / and is a command.
func (m *AISidebarModel) SlashCmdActive() bool {
	return len(m.Input) > 0 && m.Input[0] == '/' && m.State == SBSInput
}

// ProcessSlashCommand handles /commands. Returns true if handled.
func (m *AISidebarModel) ProcessSlashCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case "/models":
		m.State = SBSModels
		return true
	case "/chats":
		m.State = SBSChats
		return true
	case "/new":
		m.ClearMessages()
		m.AddMessage("system", "Started new session.")
		m.State = SBSInput
		m.Input = ""
		m.Cursor = 0
		return true
	case "/rename":
		if len(parts) > 1 {
			name := strings.Join(parts[1:], " ")
			m.SessionID = name
			m.AddMessage("system", fmt.Sprintf("Session renamed to: %s", name))
		}
		m.State = SBSInput
		m.Input = ""
		m.Cursor = 0
		return true
	}
	return false
}

// --- Key handling ---

// HandleKey processes a key press. Returns handled=true if key was consumed.
func (m *AISidebarModel) HandleKey(key string) bool {
	if !m.Active() {
		return false
	}

	// Global dismiss
	if key == "esc" {
		m.Close()
		return true
	}

	switch m.State {
	case SBSInput:
		return m.handleInputKey(key)
	case SBSModels:
		return m.handleModelsKey(key)
	case SBSChats:
		return m.handleChatsKey(key)
	case SBSProposals:
		return m.handleProposalsKey(key)
	case SBSStreaming:
		// Most keys ignored during streaming; esc cancels
		return true
	}
	return false
}

func (m *AISidebarModel) handleInputKey(key string) bool {
	switch key {
	case "enter":
		cmd := m.GetInput()
		if cmd == "" {
			return true
		}
		if m.SlashCmdActive() {
			handled := m.ProcessSlashCommand(cmd)
			m.Input = ""
			m.Cursor = 0
			return handled
		}
		// Signal that user wants to send — handled by app
		return true

	case "backspace":
		if m.Cursor > 0 {
			_, size := utf8.DecodeLastRuneInString(m.Input[:m.Cursor])
			m.Input = m.Input[:m.Cursor-size] + m.Input[m.Cursor:]
			m.Cursor -= size
		}
	case "left":
		if m.Cursor > 0 {
			_, size := utf8.DecodeLastRuneInString(m.Input[:m.Cursor])
			m.Cursor -= size
		}
	case "right":
		if m.Cursor < len(m.Input) {
			_, size := utf8.DecodeRuneInString(m.Input[m.Cursor:])
			m.Cursor += size
		}
	case "home":
		m.Cursor = 0
	case "end":
		m.Cursor = len(m.Input)
	case " ":
		m.Input = m.Input[:m.Cursor] + " " + m.Input[m.Cursor:]
		m.Cursor++
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.Input = m.Input[:m.Cursor] + key + m.Input[m.Cursor:]
			m.Cursor++
		}
	}
	return true
}

func (m *AISidebarModel) handleModelsKey(key string) bool {
	if key == "enter" || key == "esc" {
		m.State = SBSInput
		return true
	}
	return true
}

func (m *AISidebarModel) handleChatsKey(key string) bool {
	if key == "enter" || key == "esc" {
		m.State = SBSInput
		return true
	}
	return true
}

func (m *AISidebarModel) handleProposalsKey(key string) bool {
	switch key {
	case "y":
		if m.ProposalFocus >= 0 && m.ProposalFocus < len(m.Proposals) {
			p := &m.Proposals[m.ProposalFocus]
			if p.CanApprove {
				p.Status = "approved"
			}
		}
	case "n":
		if m.ProposalFocus >= 0 && m.ProposalFocus < len(m.Proposals) {
			p := &m.Proposals[m.ProposalFocus]
			if p.CanReject {
				p.Status = "rejected"
			}
		}
	case "j", "down":
		if m.ProposalFocus < len(m.Proposals)-1 {
			m.ProposalFocus++
		}
	case "k", "up":
		if m.ProposalFocus > 0 {
			m.ProposalFocus--
		}
	case "enter", "esc":
		m.State = SBSInput
	}
	return true
}

// --- View ---

func (m *AISidebarModel) View() string {
	if !m.Active() {
		return ""
	}

	panelW := m.Width
	if panelW < 20 {
		panelW = 20
	}
	contentH := m.Height
	if contentH < 5 {
		contentH = 5
	}

	var b strings.Builder

	// Header
	modeLabel := "Ask"
	if m.Mode == ModeAgent {
		modeLabel = "Agent"
	}
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Mauve).
		Render(fmt.Sprintf(" ◆ AI %s — %s", modeLabel, m.Model))
	b.WriteString(header)

	// Provider line
	if m.Provider != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.Subtext).
			Italic(true).
			Render(fmt.Sprintf("  %s", m.Provider)))
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(styles.Overlay0).
		Render(strings.Repeat("─", panelW-2)))
	b.WriteString("\n")

	// Conversation area
	availH := contentH - 7 // header + input area + borders
	if availH < 3 {
		availH = 3
	}

	// Show messages (most recent first, capped)
	msgLines := m.renderMessages()
	msgLines = m.truncateLines(msgLines, availH)

	for _, line := range msgLines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Status line for models/chats view
	switch m.State {
	case SBSModels:
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.Peach).
			Render("  Available models"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.Subtext).
			Render("  Esc to go back"))
		b.WriteString("\n")
	case SBSChats:
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.Peach).
			Render("  Chat sessions"))
		b.WriteString("\n")
		for _, s := range m.Sessions {
			b.WriteString(fmt.Sprintf("  • %s\n", s))
		}
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.Subtext).
			Render("  Esc to go back"))
		b.WriteString("\n")
	}

	// Proposals (agent mode)
	if m.State == SBSProposals && len(m.Proposals) > 0 {
		for i, p := range m.Proposals {
			focused := i == m.ProposalFocus
			statusColor := styles.Subtext
			switch p.Status {
			case "approved":
				statusColor = styles.Green
			case "rejected":
				statusColor = styles.Red
			}
			cursor := " "
			if focused {
				cursor = "▸"
			}
			b.WriteString(fmt.Sprintf("%s [%s] %s\n",
				cursor,
				lipgloss.NewStyle().Foreground(statusColor).Render(p.Status),
				p.Description,
			))
		}
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.Subtext).
			Render("  y=approve n=reject ↑↓=focus"))
		b.WriteString("\n")
	}

	// Separator
	b.WriteString(lipgloss.NewStyle().
		Foreground(styles.Overlay0).
		Render(strings.Repeat("─", panelW-2)))
	b.WriteString("\n")

	// Input area
	inputRow := "> "
	cursorChar := " "
	if m.Cursor < len(m.Input) {
		cursorChar = "█"
	}
	inputRow += m.Input[:m.Cursor] + cursorChar + m.Input[m.Cursor:]
	if len(inputRow) > panelW-4 {
		inputRow = "…" + inputRow[len(inputRow)-panelW+8:]
	}
	b.WriteString(inputRow)

	// Bottom hint
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(styles.Selection).
		Render(fmt.Sprintf("  %s — Esc to close", m.helpHint())))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(panelW).
		Padding(0, 1).
		Render(b.String())
}

func (m *AISidebarModel) helpHint() string {
	switch m.State {
	case SBSInput:
		if m.SlashCmdActive() {
			return "/models /chats /new /rename"
		}
		if m.Mode == ModeAgent {
			return "agent mode — git commands"
		}
		return "ask anything about the repo"
	case SBSStreaming:
		return "streaming response..."
	case SBSProposals:
		return "y/n to approve/reject proposals"
	default:
		return ""
	}
}

func (m *AISidebarModel) renderMessages() []string {
	var lines []string

	for _, msg := range m.Messages {
		prefix := ""
		style := lipgloss.NewStyle().Foreground(styles.Text)
		switch msg.Role {
		case "user":
			prefix = "You:"
			style = lipgloss.NewStyle().Foreground(styles.Teal).Bold(true)
		case "assistant":
			prefix = "AI:"
			style = lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true)
		case "system":
			prefix = "◆"
			style = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)
		}
		lines = append(lines, style.Render(prefix+" "+msg.Content))
	}

	// Streaming content
	if m.State == SBSStreaming && m.StreamingContent != "" {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render("AI: ")+
				lipgloss.NewStyle().Foreground(styles.Text).Render(m.StreamingContent)+
				lipgloss.NewStyle().Foreground(styles.Subtext).Render(" █"))
	}

	// Error display
	if m.Error != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.Red).Render("⚠ "+m.Error))
	}

	return lines
}

func (m *AISidebarModel) truncateLines(lines []string, max int) []string {
	if len(lines) <= max {
		return lines
	}
	// Show most recent `max-1` lines + "..."
	result := []string{lipgloss.NewStyle().
		Foreground(styles.Subtext).
		Italic(true).
		Render("  ... (older messages hidden)"),
	}
	result = append(result, lines[len(lines)-(max-1):]...)
	return result
}
