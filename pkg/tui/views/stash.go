package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// StashModel holds state for the stash list view.
type StashModel struct {
	components.ZList[*models.Stash]
	Loaded bool
	Error  string
	Prompt Prompt
}

// NewStashModel creates the stash view model.
func NewStashModel() StashModel {
	m := StashModel{}
	m.Height = 20
	m.Match = func(s *models.Stash, q string) bool {
		hay := strings.ToLower(strings.Join([]string{
			fmt.Sprintf("%d", s.Index), s.Message, s.Hash,
		}, " "))
		return strings.Contains(hay, q)
	}
	m.Render = stashRow
	return m
}

// UpdateStash refreshes the list.
func (m *StashModel) UpdateStash(items []*models.Stash) {
	m.Items = items
	m.Loaded = true
	m.Error = ""
	m.ClampCursor()
}

// View renders the stash list.
func (m *StashModel) View(width int) string {
	body := m.body(width)
	return screenChrome("Stash", screenMeta(len(m.Filtered()), len(m.Items), m.Filter), body, width)
}

// body renders everything under the screen chrome.
func (m *StashModel) body(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if !m.Loaded {
		return styles.LoadingStyle.Render("Loading stash...")
	}
	if len(m.Items) == 0 {
		return styles.SubtitleStyle.Render("Stash is empty — press s to push one")
	}

	var b strings.Builder
	b.WriteString(m.FilterBar())
	if len(m.Filtered()) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No stashes match filter")
	}
	return b.String() + m.ViewRows(width)
}

// stashRow renders one stash row (no cursor — ZList wraps it).
func stashRow(s *models.Stash, width int) string {
	hash := s.Hash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	when := ""
	if !s.Time.IsZero() {
		when = formatTime(s.Time)
	}
	msg := s.Message
	if msg == "" {
		msg = "(no message)"
	}
	line := fmt.Sprintf(" %s %s  %s %s",
		styles.SubtitleStyle.Render(fmt.Sprintf("stash@{%d}", s.Index)),
		truncateStr(msg, 44), styles.SubtitleStyle.Render(hash),
		styles.SubtitleStyle.Render(when))
	if lipgloss.Width(line) > width {
		line = fmt.Sprintf(" %s %s",
			styles.SubtitleStyle.Render(fmt.Sprintf("stash@{%d}", s.Index)),
			truncateStr(msg, width-14))
	}
	return line
}
