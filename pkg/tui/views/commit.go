package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// CommitDialogState tracks the commit compose flow.
type CommitDialogState int

const (
	CommitInputSubject CommitDialogState = iota
	CommitInputBody
	CommitConfirming
	CommitResult
)

// CommitResultInfo holds the outcome of a commit.
type CommitResultInfo struct {
	Success bool
	Hash    string
	Message string
	Error   string
}

// CommitModel is an interactive commit message dialog.
type CommitModel struct {
	Subject textinput.Model
	Body    textarea.Model
	State   CommitDialogState
	Result  CommitResultInfo
	width   int
	height  int
	active  bool
}

// NewCommitModel creates a new commit dialog.
func NewCommitModel() CommitModel {
	subject := textinput.New()
	subject.Placeholder = "Commit summary (required)"
	subject.Focus()
	subject.CharLimit = 72
	subject.Width = 60
	subject.Prompt = "─ "
	subject.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	subject.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)

	body := textarea.New()
	body.Placeholder = "Optional detailed description (Ctrl+D to finish)"
	body.CharLimit = 0
	body.SetWidth(60)
	body.SetHeight(8)
	body.ShowLineNumbers = false
	body.Prompt = ""
	body.FocusedStyle.Text = lipgloss.NewStyle().Foreground(styles.Text)
	body.BlurredStyle.Text = lipgloss.NewStyle().Foreground(styles.Text)
	body.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)
	body.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)

	return CommitModel{
		Subject: subject,
		Body:    body,
		State:   CommitInputSubject,
		active:  true,
	}
}

// Active returns true when the commit dialog is open.
func (m *CommitModel) Active() bool {
	return m.active
}

// Open activates the dialog and resets state.
func (m *CommitModel) Open() {
	m.active = true
	m.State = CommitInputSubject
	m.Subject.SetValue("")
	m.Body.SetValue("")
	m.Subject.Focus()
	m.Result = CommitResultInfo{}
}

// Close deactivates the dialog.
func (m *CommitModel) Close() {
	m.active = false
	m.Subject.Blur()
	m.Body.Blur()
}

// Init implements tea.Model.
func (m CommitModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key events for the commit dialog.
func (m *CommitModel) Update(msg tea.Msg) (*CommitModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Subject.Width = msg.Width - 20
		m.Body.SetWidth(msg.Width - 20)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel at any state
			m.Close()
			return m, nil

		case "ctrl+d":
			// Finish body input
			if m.State == CommitInputBody {
				m.State = CommitConfirming
				return m, nil
			}

		case "enter":
			switch m.State {
			case CommitInputSubject:
				subject := strings.TrimSpace(m.Subject.Value())
				if subject == "" {
					m.Result = CommitResultInfo{
						Success: false,
						Error:   "Commit message cannot be empty",
					}
					m.State = CommitResult
					return m, nil
				}
				// Move to body
				m.State = CommitInputBody
				m.Subject.Blur()
				m.Body.Focus()
				return m, nil
			}
			// CommitConfirming + enter handled at app level
		}
	}

	// Delegate to active input
	var cmd tea.Cmd
	switch m.State {
	case CommitInputSubject:
		updatedSubject, subjectCmd := m.Subject.Update(msg)
		m.Subject = updatedSubject
		cmd = subjectCmd
	case CommitInputBody:
		updatedBody, bodyCmd := m.Body.Update(msg)
		m.Body = updatedBody
		cmd = bodyCmd
	}

	return m, cmd
}

// View renders the commit dialog overlay.
func (m *CommitModel) View(width int) string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	b.WriteString(styles.DialogBoxStyle.Render(m.renderContent(width)))
	return b.String()
}

// renderContent returns the inner content of the commit dialog.
func (m *CommitModel) renderContent(width int) string {
	var b strings.Builder

	b.WriteString(styles.DialogTitleStyle.Render(" Commit Changes"))
	b.WriteString("\n\n")

	switch m.State {
	case CommitInputSubject:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Summary (required):"))
		b.WriteString("\n")
		b.WriteString(m.Subject.View())
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to continue, Esc to cancel"))

	case CommitInputBody:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Summary:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.Subject.Value()))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description (optional):"))
		b.WriteString("\n")
		b.WriteString(m.Body.View())
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Ctrl+D to finish, Esc to cancel"))

	case CommitConfirming:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Summary:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.Subject.Value()))
		b.WriteString("\n\n")
		if body := strings.TrimSpace(m.Body.Value()); body != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + body))
			b.WriteString("\n\n")
		}
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to confirm commit, Esc to cancel"))

	case CommitResult:
		if m.Result.Success {
			b.WriteString(styles.StatusStagedStyle.Render(" ✓ Commit successful"))
			if m.Result.Hash != "" {
				b.WriteString("\n")
				b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render("  " + m.Result.Hash))
			}
		} else {
			b.WriteString(styles.ErrorStyle.Render(" ✗ Commit failed"))
			if m.Result.Error != "" {
				b.WriteString("\n")
				b.WriteString(lipgloss.NewStyle().Foreground(styles.Red).Render("  " + m.Result.Error))
			}
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter or Esc to close"))
	}

	return b.String()
}

// GetMessage returns the composed commit message (subject + body).
func (m *CommitModel) GetMessage() string {
	subject := strings.TrimSpace(m.Subject.Value())
	body := strings.TrimSpace(m.Body.Value())
	if body == "" {
		return subject
	}
	return fmt.Sprintf("%s\n\n%s", subject, body)
}
