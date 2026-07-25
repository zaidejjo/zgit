package views

import (
	"fmt"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// IssueListModel holds state for the issue list view.
type IssueListModel struct {
	Issues []*models.Issue
	Cursor int
	Error  string
	Height int
}

// NewIssueListModel creates a default issue list view model.
func NewIssueListModel() IssueListModel {
	return IssueListModel{}
}

// UpdateIssues refreshes the issue list data.
func (m *IssueListModel) UpdateIssues(issues []*models.Issue) {
	m.Issues = issues
	m.Error = ""
	if m.Cursor >= len(issues) {
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

	b.WriteString(styles.SectionTitleStyle.Render(fmt.Sprintf(" Issues (%d)", len(m.Issues))))
	b.WriteString("\n")

	for i, issue := range m.Issues {
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
