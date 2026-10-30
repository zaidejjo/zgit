package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// IssueListModel holds state for the issue list view.
type IssueListModel struct {
	components.ZList[*models.Issue]
	Error string
}

// NewIssueListModel creates a default issue list view model.
func NewIssueListModel() IssueListModel {
	m := IssueListModel{}
	m.Height = 20
	m.Match = func(issue *models.Issue, q string) bool {
		hay := strings.ToLower(strings.Join([]string{
			fmt.Sprintf("%d", issue.Number), issue.Title, issue.Author,
		}, " "))
		return strings.Contains(hay, q)
	}
	m.Render = issueRow
	return m
}

// UpdateIssues refreshes the issue list data.
func (m *IssueListModel) UpdateIssues(issues []*models.Issue) {
	m.Items = issues
	m.Error = ""
	m.ClampCursor()
}

// View renders the issue list.
func (m *IssueListModel) View(width int) string {
	body := m.body(width)
	return screenChrome("Issues", screenMeta(len(m.Filtered()), len(m.Items), m.Filter), body, width)
}

// body renders everything under the screen chrome.
func (m *IssueListModel) body(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if m.Items == nil {
		return styles.LoadingStyle.Render("Loading issues...")
	}

	if len(m.Items) == 0 {
		return styles.SubtitleStyle.Render("No issues found")
	}

	var b strings.Builder
	b.WriteString(m.FilterBar())
	if len(m.Filtered()) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No issues match filter")
	}

	return b.String() + m.ViewRows(width)
}

// issueRow renders one issue row (no cursor affordance — ZList wraps it).
func issueRow(issue *models.Issue, width int) string {
	stateMark := issueStateMark(issue.State)
	timeAgo := formatTime(issue.CreatedAt)
	labels := formatLabels(issue.Labels)

	title := fmt.Sprintf("#%d %s", issue.Number, truncateStr(issue.Title, width-22))
	meta := fmt.Sprintf("  %s %s%s  %s  by %s",
		stateMark, title, labels, timeAgo, issue.Author)
	if lipgloss.Width(meta) > width {
		title = fmt.Sprintf("#%d %s", issue.Number, truncateStr(issue.Title, width-30))
		meta = fmt.Sprintf("  %s %s  %s", stateMark, title, timeAgo)
	}
	return meta
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
