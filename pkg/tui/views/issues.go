package views

import (
	"fmt"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// IssueListModel holds state for the issue list view.
type IssueListModel struct {
	Issues    []*models.Issue
	Cursor    int
	Error     string
	Height    int
	Filter    string
	Filtering bool
}

// NewIssueListModel creates a default issue list view model.
func NewIssueListModel() IssueListModel {
	return IssueListModel{}
}

// UpdateIssues refreshes the issue list data.
func (m *IssueListModel) UpdateIssues(issues []*models.Issue) {
	m.Issues = issues
	m.Error = ""
	m.ClampCursor()
}

// Filtered returns issues matching the current filter (case-insensitive
// substring over number, title and author).
func (m *IssueListModel) Filtered() []*models.Issue {
	q := strings.ToLower(strings.TrimSpace(m.Filter))
	if q == "" {
		return m.Issues
	}
	out := make([]*models.Issue, 0, len(m.Issues))
	for _, issue := range m.Issues {
		hay := strings.ToLower(strings.Join([]string{
			fmt.Sprintf("%d", issue.Number), issue.Title, issue.Author,
		}, " "))
		if strings.Contains(hay, q) {
			out = append(out, issue)
		}
	}
	return out
}

// ClampCursor keeps the cursor inside the filtered list.
func (m *IssueListModel) ClampCursor() {
	if m.Cursor >= len(m.Filtered()) {
		m.Cursor = 0
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

// View renders the issue list.
func (m *IssueListModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if m.Issues == nil {
		return styles.LoadingStyle.Render("Loading issues...")
	}

	if len(m.Issues) == 0 {
		return styles.SubtitleStyle.Render("No issues found")
	}

	var b strings.Builder

	m.ClampCursor()

	title := fmt.Sprintf(" Issues (%d)", len(m.Issues))
	if strings.TrimSpace(m.Filter) != "" {
		title = fmt.Sprintf(" Issues (%d/%d) filter: %s", len(m.Filtered()), len(m.Issues), m.Filter)
	}
	b.WriteString(styles.SectionTitleStyle.Render(title))
	b.WriteString("\n")
	if m.Filtering {
		b.WriteString(styles.FilterBarStyle.Render(" /" + m.Filter + "\u2588"))
		b.WriteString("\n")
	}

	visible := m.Filtered()
	if len(visible) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No issues match filter")
	}

	for i, issue := range visible {
		if i < m.Cursor-m.Height || i > m.Cursor+m.Height {
			continue
		}

		stateMark := issueStateMark(issue.State)
		timeAgo := formatTime(issue.CreatedAt)
		labels := formatLabels(issue.Labels)

		title := fmt.Sprintf("#%d %s", issue.Number, truncateStr(issue.Title, width-22))
		meta := fmt.Sprintf("  %s %s%s  %s  by %s",
			stateMark, title, labels, timeAgo, issue.Author)

		if i == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render(meta))
		} else {
			b.WriteString(styles.ListItemStyle.Render(meta))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// issueStateMark returns a visual indicator for issue state.
func issueStateMark(state models.IssueState) string {
	switch state {
	case models.IssueOpen:
		return styles.StatusStagedStyle.Render("●")
	case models.IssueClosed:
		return styles.StatusDeletedStyle.Render("●")
	default:
		return "○"
	}
}

// formatLabels returns a space-separated string of label badges.
func formatLabels(labels []models.Label) string {
	if len(labels) == 0 {
		return ""
	}
	var b strings.Builder
	for _, l := range labels {
		b.WriteString(" ")
		s := styles.ListItemStyle.Copy().
			Background(styles.Selection).
			Foreground(styles.Subtext).
			Render(l.Name)
		b.WriteString(s)
	}
	return b.String()
}
