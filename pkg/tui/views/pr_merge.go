package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// PRMergeState tracks the PR merge flow.
type PRMergeState int

const (
	PRMergeSelectStrategy PRMergeState = iota
	PRMergeConfirming
	PRMergeResult
)

// PRMergeResultInfo holds the outcome of a PR merge.
type PRMergeResultInfo struct {
	Success bool
	Message string
	Error   string
}

// PRMergeModel is a pull request merge confirmation dialog.
type PRMergeModel struct {
	PR       *models.PullRequestSummary
	State    PRMergeState
	Strategy MergeStrategy
	Result   PRMergeResultInfo
	width    int
	height   int
	active   bool
}

// NewPRMergeModel creates a new PR merge dialog.
func NewPRMergeModel() PRMergeModel {
	return PRMergeModel{
		Strategy: MergeNormal,
	}
}

// Active returns true when the dialog is open.
func (m *PRMergeModel) Active() bool {
	return m.active
}

// Open activates the dialog for the given PR.
func (m *PRMergeModel) Open(pr *models.PullRequestSummary) {
	m.active = true
	m.PR = pr
	m.State = PRMergeSelectStrategy
	m.Strategy = MergeNormal
	m.Result = PRMergeResultInfo{}
}

// Close deactivates the dialog.
func (m *PRMergeModel) Close() {
	m.active = false
	m.PR = nil
}

// Update handles key events for the PR merge dialog.
func (m *PRMergeModel) Update(msg tea.Msg) (*PRMergeModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Close()
			return m, nil

		case "enter":
			if m.State == PRMergeSelectStrategy {
				m.State = PRMergeConfirming
				return m, nil
			}
			// PRMergeConfirming + enter handled at app level

		case "up", "left":
			if m.State == PRMergeSelectStrategy {
				if m.Strategy > MergeNormal {
					m.Strategy--
				}
				return m, nil
			}

		case "down", "right":
			if m.State == PRMergeSelectStrategy {
				if m.Strategy < MergeRebase {
					m.Strategy++
				}
				return m, nil
			}
		}
	}

	return m, nil
}

// View renders the PR merge dialog overlay.
func (m *PRMergeModel) View(width int) string {
	if !m.active || m.PR == nil {
		return ""
	}

	return styles.DialogBoxStyle.Render(m.renderContent(width))
}

func (m *PRMergeModel) renderContent(width int) string {
	var b strings.Builder

	b.WriteString(styles.DialogTitleStyle.Render(" Merge Pull Request"))
	b.WriteString("\n\n")

	switch m.State {
	case PRMergeSelectStrategy:
		b.WriteString(fmt.Sprintf("  #%d %s", m.PR.Number, m.PR.Title))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf("  %s → %s", m.PR.HeadRef, m.PR.BaseRef)))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf("  by %s", m.PR.Author)))
		if m.PR.Mergeable != "" {
			b.WriteString("\n")
			mergeableStr := "Unknown"
			mergeableStyle := lipgloss.NewStyle().Foreground(styles.Subtext)
			if m.PR.Mergeable == "MERGEABLE" {
				mergeableStr = "Mergeable"
				mergeableStyle = styles.StatusStagedStyle
			} else if m.PR.Mergeable == "CONFLICTING" {
				mergeableStr = "Has conflicts"
				mergeableStyle = styles.ErrorStyle
			}
			b.WriteString(mergeableStyle.Render("  " + mergeableStr))
		}
		b.WriteString("\n\n")
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

	case PRMergeConfirming:
		b.WriteString(fmt.Sprintf("  #%d %s", m.PR.Number, m.PR.Title))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(fmt.Sprintf("  %s → %s", m.PR.HeadRef, m.PR.BaseRef)))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Merge Strategy: "))
		b.WriteString(styles.ListItemActiveStyle.Render(m.Strategy.String()))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Press Enter to merge, Esc to cancel"))

	case PRMergeResult:
		if m.Result.Success {
			b.WriteString(styles.StatusStagedStyle.Render(" ✓ PR merged successfully"))
			if m.Result.Message != "" {
				b.WriteString("\n")
				b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render("  " + m.Result.Message))
			}
		} else {
			b.WriteString(styles.ErrorStyle.Render(" ✗ PR merge failed"))
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
