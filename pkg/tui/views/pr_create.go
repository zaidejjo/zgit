package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// PRCreateState tracks the PR creation flow.
type PRCreateState int

const (
	// PRCreateForm is the huh form (title + body + strategy).
	PRCreateForm PRCreateState = iota
	// PRCreateConfirming shows the final review before creating.
	PRCreateConfirming
	// PRCreateResult shows the outcome.
	PRCreateResult
)

// MergeStrategy for pull request merging.
type MergeStrategy int

const (
	MergeNormal MergeStrategy = iota
	MergeSquash
	MergeRebase
)

func (s MergeStrategy) String() string {
	switch s {
	case MergeNormal:
		return "Create Merge Commit"
	case MergeSquash:
		return "Squash & Merge"
	case MergeRebase:
		return "Rebase & Merge"
	}
	return "Create Merge Commit"
}

func (s MergeStrategy) APIValue() string {
	switch s {
	case MergeNormal:
		return "merge"
	case MergeSquash:
		return "squash"
	case MergeRebase:
		return "rebase"
	}
	return "merge"
}

// ValidatePRTitle rejects empty and overlong titles.
func ValidatePRTitle(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("title is required")
	}
	if len([]rune(s)) > 120 {
		return fmt.Errorf("keep the title under 120 chars")
	}
	return nil
}

// PRCreateResultInfo holds the outcome of a PR creation.
type PRCreateResultInfo struct {
	Success bool
	Number  int
	URL     string
	Error   string
}

// PRCreateModel is a huh-backed pull request creation dialog.
type PRCreateModel struct {
	form       *huh.Form
	title      string
	body       string
	Strategy   MergeStrategy // selected strategy (synced on complete)
	HeadBranch string        // source branch (pre-filled)
	BaseBranch string        // target branch (pre-filled)
	State      PRCreateState
	Result     PRCreateResultInfo
	width      int
	height     int
	active     bool
	needInit   bool
}

// NewPRCreateModel creates a new PR creation dialog.
func NewPRCreateModel() PRCreateModel {
	m := PRCreateModel{State: PRCreateForm, active: true}
	m.buildForm()
	m.needInit = true
	return m
}

func (m *PRCreateModel) buildForm() {
	strategyLabel := m.Strategy.String()
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Description(fmt.Sprintf("Branch: %s → %s", m.HeadBranch, m.BaseBranch)).
				Placeholder("PR title (required)").
				CharLimit(120).
				Value(&m.title).
				Validate(ValidatePRTitle),
			huh.NewText().
				Title("Description").
				Placeholder("Optional description").
				CharLimit(4000).
				Lines(5).
				Value(&m.body),
			huh.NewSelect[string]().
				Title("Merge strategy").
				Key("strategy").
				Options(
					huh.NewOption(MergeNormal.String(), MergeNormal.String()),
					huh.NewOption(MergeSquash.String(), MergeSquash.String()),
					huh.NewOption(MergeRebase.String(), MergeRebase.String()),
				).
				Value(&strategyLabel).
				Description("How this PR will merge"),
		),
	).WithShowHelp(true)
	if m.width > 0 {
		m.form = m.form.WithWidth(m.width)
	}
	m.syncStrategy(strategyLabel)
}

// syncStrategy maps the select label back to the enum.
func (m *PRCreateModel) syncStrategy(label string) {
	switch label {
	case MergeSquash.String():
		m.Strategy = MergeSquash
	case MergeRebase.String():
		m.Strategy = MergeRebase
	default:
		m.Strategy = MergeNormal
	}
}

// Active returns true when the dialog is open.
func (m *PRCreateModel) Active() bool {
	return m.active
}

// Open activates the dialog with pre-filled branch info.
func (m *PRCreateModel) Open(headBranch, baseBranch string) {
	m.active = true
	m.State = PRCreateForm
	m.HeadBranch = headBranch
	m.BaseBranch = baseBranch
	m.Strategy = MergeNormal
	m.title = ""
	m.body = ""
	m.Result = PRCreateResultInfo{}
	m.buildForm()
	m.needInit = true
}

// Close deactivates the dialog.
func (m *PRCreateModel) Close() {
	m.active = false
}

// Init implements tea.Model.
func (m PRCreateModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the PR creation dialog.
func (m *PRCreateModel) Update(msg tea.Msg) (*PRCreateModel, tea.Cmd) {
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
		if m.form != nil {
			m.form = m.form.WithWidth(msg.Width - 4)
		}
		return m, nil

	case tea.KeyMsg:
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
		m.syncStrategy(m.form.GetString("strategy"))
		m.State = PRCreateConfirming
		return m, nil
	}
	if m.form.State == huh.StateAborted {
		m.Close()
		return m, nil
	}
	return m, cmd
}

// GetRequest returns the PR request for the API call.
func (m *PRCreateModel) GetRequest() github.PRRequest {
	return github.PRRequest{
		Title: strings.TrimSpace(m.title),
		Body:  strings.TrimSpace(m.body),
		Head:  m.HeadBranch,
		Base:  m.BaseBranch,
		Draft: false,
	}
}

// GetMergeStrategy returns the selected merge strategy API value.
func (m *PRCreateModel) GetMergeStrategy() string {
	return m.Strategy.APIValue()
}

// View renders the PR creation dialog overlay.
func (m *PRCreateModel) View(width int) string {
	if !m.active {
		return ""
	}

	var b strings.Builder
	b.WriteString(styles.DialogTitleStyle.Render(" Create Pull Request"))
	b.WriteString("\n\n")
	switch m.State {
	case PRCreateForm:
		if m.form != nil {
			b.WriteString(m.form.View())
		}
	case PRCreateConfirming:
		b.WriteString(m.renderConfirm())
	case PRCreateResult:
		b.WriteString(m.renderResult())
	}
	return styles.DialogBoxStyle.Render(b.String())
}

// renderConfirm shows the final review before creating.
func (m *PRCreateModel) renderConfirm() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf(" Branch: %s → %s", m.HeadBranch, m.BaseBranch)))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Title:"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.title))
	b.WriteString("\n\n")
	if body := strings.TrimSpace(m.body); body != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + body))
		b.WriteString("\n\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Merge Strategy: "))
	b.WriteString(styles.ListItemActiveStyle.Render(m.Strategy.String()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to create PR, Esc to cancel"))
	return b.String()
}

// renderResult shows the PR creation outcome.
func (m *PRCreateModel) renderResult() string {
	var b strings.Builder
	if m.Result.Success {
		b.WriteString(styles.StatusStagedStyle.Render(fmt.Sprintf(" ✓ PR #%d created", m.Result.Number)))
		if m.Result.URL != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render("  " + m.Result.URL))
		}
	} else {
		b.WriteString(styles.ErrorStyle.Render(" ✗ PR creation failed"))
		if m.Result.Error != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Red).Render("  " + m.Result.Error))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter or Esc to close"))
	return b.String()
}
