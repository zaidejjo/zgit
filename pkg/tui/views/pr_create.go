package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// PRCreateState tracks the PR creation flow.
type PRCreateState int

const (
	PRCreateInputTitle PRCreateState = iota
	PRCreateInputBody
	PRCreateSelectStrategy
	PRCreateConfirming
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

// PRCreateResultInfo holds the outcome of a PR creation.
type PRCreateResultInfo struct {
	Success bool
	Number  int
	URL     string
	Error   string
}

// PRCreateModel is an interactive pull request creation dialog.
type PRCreateModel struct {
	Title      textinput.Model
	Body       textarea.Model
	State      PRCreateState
	Strategy   MergeStrategy
	HeadBranch string // source branch (pre-filled)
	BaseBranch string // target branch (pre-filled)
	Result     PRCreateResultInfo
	width      int
	height     int
	active     bool
}

// NewPRCreateModel creates a new PR creation dialog.
func NewPRCreateModel() PRCreateModel {
	title := textinput.New()
	title.Placeholder = "PR title (required)"
	title.Focus()
	title.CharLimit = 120
	title.Width = 60
	title.Prompt = "─ "
	title.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	title.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)

	body := textarea.New()
	body.Placeholder = "Optional description (Ctrl+D to finish)"
	body.CharLimit = 0
	body.SetWidth(60)
	body.SetHeight(8)
	body.ShowLineNumbers = false
	body.Prompt = ""
	body.FocusedStyle.Text = lipgloss.NewStyle().Foreground(styles.Text)
	body.BlurredStyle.Text = lipgloss.NewStyle().Foreground(styles.Text)
	body.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)
	body.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true)

	return PRCreateModel{
		Title:    title,
		Body:     body,
		State:    PRCreateInputTitle,
		Strategy: MergeNormal,
		active:   true,
	}
}

// Active returns true when the dialog is open.
func (m *PRCreateModel) Active() bool {
	return m.active
}

// Open activates the dialog with pre-filled branch info.
func (m *PRCreateModel) Open(headBranch, baseBranch string) {
	m.active = true
	m.State = PRCreateInputTitle
	m.HeadBranch = headBranch
	m.BaseBranch = baseBranch
	m.Strategy = MergeNormal
	m.Title.SetValue("")
	m.Body.SetValue("")
	m.Title.Focus()
	m.Result = PRCreateResultInfo{}
}

// Close deactivates the dialog.
func (m *PRCreateModel) Close() {
	m.active = false
	m.Title.Blur()
	m.Body.Blur()
}

// Init implements tea.Model.
func (m PRCreateModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key events for the PR creation dialog.
func (m *PRCreateModel) Update(msg tea.Msg) (*PRCreateModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Title.Width = msg.Width - 20
		m.Body.SetWidth(msg.Width - 20)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Close()
			return m, nil

		case "ctrl+d":
			if m.State == PRCreateInputBody {
				m.State = PRCreateSelectStrategy
				m.Title.Blur()
				m.Body.Blur()
				return m, nil
			}

		case "enter":
			switch m.State {
			case PRCreateInputTitle:
				title := strings.TrimSpace(m.Title.Value())
				if title == "" {
					m.Result = PRCreateResultInfo{
						Success: false,
						Error:   "PR title cannot be empty",
					}
					m.State = PRCreateResult
					return m, nil
				}
				m.State = PRCreateInputBody
				m.Title.Blur()
				m.Body.Focus()
				return m, nil

			case PRCreateSelectStrategy:
				// Enter confirms the strategy selection
				m.State = PRCreateConfirming
				return m, nil
			}

		case "up", "left":
			if m.State == PRCreateSelectStrategy {
				if m.Strategy > MergeNormal {
					m.Strategy--
				}
				return m, nil
			}

		case "down", "right":
			if m.State == PRCreateSelectStrategy {
				if m.Strategy < MergeRebase {
					m.Strategy++
				}
				return m, nil
			}
		}
	}

	// Delegate to active input
	var cmd tea.Cmd
	switch m.State {
	case PRCreateInputTitle:
		updated, titleCmd := m.Title.Update(msg)
		m.Title = updated
		cmd = titleCmd
	case PRCreateInputBody:
		updated, bodyCmd := m.Body.Update(msg)
		m.Body = updated
		cmd = bodyCmd
	}

	return m, cmd
}

// GetRequest returns the PR request for the API call.
func (m *PRCreateModel) GetRequest() github.PRRequest {
	return github.PRRequest{
		Title: strings.TrimSpace(m.Title.Value()),
		Body:  strings.TrimSpace(m.Body.Value()),
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

	return styles.DialogBoxStyle.Render(m.renderContent(width))
}

func (m *PRCreateModel) renderContent(width int) string {
	var b strings.Builder

	b.WriteString(styles.DialogTitleStyle.Render(" Create Pull Request"))
	b.WriteString("\n\n")

	switch m.State {
	case PRCreateInputTitle:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf(" Branch: %s → %s", m.HeadBranch, m.BaseBranch)))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Title (required):"))
		b.WriteString("\n")
		b.WriteString(m.Title.View())
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to continue, Esc to cancel"))

	case PRCreateInputBody:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf(" Branch: %s → %s", m.HeadBranch, m.BaseBranch)))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Title:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.Title.Value()))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description (optional):"))
		b.WriteString("\n")
		b.WriteString(m.Body.View())
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Ctrl+D to continue, Esc to cancel"))

	case PRCreateSelectStrategy:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Title:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.Title.Value()))
		b.WriteString("\n\n")
		if body := strings.TrimSpace(m.Body.Value()); body != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + body))
			b.WriteString("\n\n")
		}
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Merge Strategy:"))
		b.WriteString("\n")
		for _, s := range []MergeStrategy{MergeNormal, MergeSquash, MergeRebase} {
			mark := "  "
			style := lipgloss.NewStyle().Foreground(styles.Subtext)
			if s == m.Strategy {
				mark = "❯ "
				style = lipgloss.NewStyle().Foreground(styles.Blue).Bold(true)
			}
			b.WriteString(fmt.Sprintf("  %s%s\n", mark, style.Render(s.String())))
		}
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" ↑↓ to select, Enter to confirm, Esc to cancel"))

	case PRCreateConfirming:
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf(" Branch: %s → %s", m.HeadBranch, m.BaseBranch)))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Title:"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + m.Title.Value()))
		b.WriteString("\n\n")
		if body := strings.TrimSpace(m.Body.Value()); body != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Description:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Text).Render("  " + body))
			b.WriteString("\n\n")
		}
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Merge Strategy: "))
		b.WriteString(styles.ListItemActiveStyle.Render(m.Strategy.String()))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to create PR, Esc to cancel"))

	case PRCreateResult:
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
	}

	return b.String()
}
