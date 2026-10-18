package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// CommitDialogState tracks the commit compose flow.
type CommitDialogState int

const (
	// CommitForm is the huh form (subject + body with validation).
	CommitForm CommitDialogState = iota
	// CommitConfirming shows the final review before executing.
	CommitConfirming
	// CommitResult shows the outcome.
	CommitResult
)

// CommitResultInfo holds the outcome of a commit.
type CommitResultInfo struct {
	Success bool
	Hash    string
	Message string
	Error   string
}

// CommitModel is a huh-backed commit message dialog.
// The form owns input navigation and validation; the app owns
// confirmation execution and result display (unchanged contract).
type CommitModel struct {
	form     *huh.Form
	subject  string
	body     string
	State    CommitDialogState
	Result   CommitResultInfo
	width    int
	height   int
	active   bool
	needInit bool
}

// ValidateCommitSubject rejects empty and overlong subjects.
func ValidateCommitSubject(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("summary is required")
	}
	if len([]rune(s)) > 72 {
		return fmt.Errorf("keep the summary under 72 chars")
	}
	return nil
}

// NewCommitModel creates a new commit dialog.
func NewCommitModel() CommitModel {
	m := CommitModel{State: CommitForm, active: true}
	m.buildForm()
	m.needInit = true
	return m
}

func (m *CommitModel) buildForm() {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Summary").
				Description("One line, imperative mood").
				Placeholder("Commit summary (required)").
				CharLimit(72).
				Value(&m.subject).
				Validate(ValidateCommitSubject),
			huh.NewText().
				Title("Description").
				Description("Optional details").
				Placeholder("What changed and why").
				CharLimit(2000).
				Lines(5).
				Value(&m.body),
		),
	).WithShowHelp(true)
	if m.width > 0 {
		_, formW := dialogWidths(m.width)
		m.form = m.form.WithWidth(formW)
	}
}

// Active returns true when the commit dialog is open.
func (m *CommitModel) Active() bool {
	return m.active
}

// Open activates the dialog and resets state.
func (m *CommitModel) Open() {
	m.active = true
	m.State = CommitForm
	m.subject = ""
	m.body = ""
	m.Result = CommitResultInfo{}
	m.buildForm()
	m.needInit = true
}

// SetAIMessage prefills the form (e.g. AI-generated message).
func (m *CommitModel) SetAIMessage(subject, body string) {
	m.subject = subject
	m.body = body
	m.buildForm()
	m.needInit = true
}

// Close deactivates the dialog.
func (m *CommitModel) Close() {
	m.active = false
}

// Init implements tea.Model.
func (m CommitModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the commit dialog.
func (m *CommitModel) Update(msg tea.Msg) (*CommitModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	if m.needInit {
		m.needInit = false
		if m.form != nil {
			return m, m.form.Init()
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Esc always cancels the dialog (huh never sees it).
		if msg.String() == "esc" {
			m.Close()
			return m, nil
		}
	}

	if m.form == nil {
		return m, nil
	}
	f, cmd := m.form.Update(msg)
	if nf, ok := f.(*huh.Form); ok {
		m.form = nf
	}
	if m.form.State == huh.StateCompleted {
		m.State = CommitConfirming
		return m, nil
	}
	if m.form.State == huh.StateAborted {
		m.Close()
		return m, nil
	}
	return m, cmd
}

// View renders the commit dialog overlay.
func (m *CommitModel) View(width int) string {
	if !m.active {
		return ""
	}

	// Size the form here rather than at construction/activation: the
	// terminal may resize (or the dialog may open before the first
	// WindowSizeMsg) and a stale width is what clipped the box.
	content, formW := dialogWidths(width)
	if m.form != nil && m.State == CommitForm {
		m.form = m.form.WithWidth(formW)
	}

	var b strings.Builder
	b.WriteString(styles.DialogTitleStyle.Render(" Commit Changes"))
	b.WriteString("\n\n")
	switch {
	case m.State == CommitResult:
		b.WriteString(m.renderResult())
	case m.State == CommitConfirming:
		b.WriteString(m.renderConfirm())
	case m.form != nil:
		b.WriteString(m.form.View())
	}
	return styles.DialogBoxStyle.Render(clampBlock(b.String(), content))
}

// renderConfirm shows the final review before executing.
func (m *CommitModel) renderConfirm() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Summary:"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.subject))
	b.WriteString("\n\n")
	if body := strings.TrimSpace(m.body); body != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + body))
		b.WriteString("\n\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to confirm commit, Esc to cancel"))
	return b.String()
}

// renderResult shows the commit outcome.
func (m *CommitModel) renderResult() string {
	var b strings.Builder
	if m.Result.Success {
		b.WriteString(styles.StatusStagedStyle.Render(" ✓ Commit successful"))
		if m.Result.Hash != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render("  " + m.Result.Hash))
		}
		if m.Result.Error != "" {
			b.WriteString("\n")
			b.WriteString(styles.ErrorStyle.Render(" ⚠ " + m.Result.Error))
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
	return b.String()
}

// GetMessage returns the composed commit message (subject + body).
func (m *CommitModel) GetMessage() string {
	subject := strings.TrimSpace(m.subject)
	body := strings.TrimSpace(m.body)
	if body == "" {
		return subject
	}
	return fmt.Sprintf("%s\n\n%s", subject, body)
}
