package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// WorktreeModel holds state for the worktree list view.
type WorktreeModel struct {
	components.ZList[*models.Worktree]
	Loaded bool
	Error  string
	Prompt Prompt
}

// NewWorktreeModel creates the worktree view model.
func NewWorktreeModel() WorktreeModel {
	m := WorktreeModel{}
	m.Height = 20
	m.Match = func(w *models.Worktree, q string) bool {
		hay := strings.ToLower(strings.Join([]string{w.Path, w.Branch, w.Head}, " "))
		return strings.Contains(hay, q)
	}
	m.Render = worktreeRow
	return m
}

// UpdateWorktrees refreshes the list.
func (m *WorktreeModel) UpdateWorktrees(items []*models.Worktree) {
	m.Items = items
	m.Loaded = true
	m.Error = ""
	m.ClampCursor()
}

// View renders the worktree list.
func (m *WorktreeModel) View(width int) string {
	body := m.body(width)
	return screenChrome("Worktrees", screenMeta(len(m.Filtered()), len(m.Items), m.Filter), body, width)
}

// body renders everything under the screen chrome.
func (m *WorktreeModel) body(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if !m.Loaded {
		return styles.LoadingStyle.Render("Loading worktrees...")
	}
	if len(m.Items) == 0 {
		return styles.SubtitleStyle.Render("No worktrees — press a to add one")
	}

	var b strings.Builder
	b.WriteString(m.FilterBar())
	if len(m.Filtered()) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No worktrees match filter")
	}
	return b.String() + m.ViewRows(width)
}

// worktreeRow renders one worktree row (no cursor — ZList wraps it).
func worktreeRow(w *models.Worktree, width int) string {
	mark := styles.StatusStagedStyle.Render("●")
	if w.IsBare {
		mark = styles.SubtitleStyle.Render("○")
	}
	name := w.Path
	if w.IsMain {
		name += styles.EmphasisStyle.Render(" ‹main›")
	}
	detail := w.Branch
	if detail == "" {
		if len(w.Head) >= 7 {
			detail = "detached " + w.Head[:7]
		} else {
			detail = "bare"
		}
	}
	if w.Locked {
		detail += styles.ErrorStyle.Render(" locked")
	}
	if w.Prunable {
		detail += styles.SubtitleStyle.Render(" prunable")
	}
	line := fmt.Sprintf(" %s %s  %s", mark, truncateStr(name, 44),
		styles.SubtitleStyle.Render(truncateStr(detail, 30)))
	if lipgloss.Width(line) > width {
		line = fmt.Sprintf(" %s %s", mark, truncateStr(name, width-6))
	}
	return line
}
