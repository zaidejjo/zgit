package views

import (
	"fmt"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// PRListModel holds state for the pull request list view.
type PRListModel struct {
	PRs    []*models.PullRequestSummary
	Cursor int
	Error  string
	Height int
}

// NewPRListModel creates a default PR list view model.
func NewPRListModel() PRListModel {
	return PRListModel{}
}

// UpdatePRs refreshes the PR list data.
func (m *PRListModel) UpdatePRs(prs []*models.PullRequestSummary) {
	m.PRs = prs
	m.Error = ""
	if m.Cursor >= len(prs) {
		m.Cursor = 0
	}
}

// View renders the PR list.
func (m *PRListModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if m.PRs == nil {
		return styles.LoadingStyle.Render("Loading pull requests...")
	}

	if len(m.PRs) == 0 {
		return styles.SubtitleStyle.Render("No pull requests found")
	}

	var b strings.Builder

	b.WriteString(styles.SectionTitleStyle.Render(fmt.Sprintf(" Pull Requests (%d)", len(m.PRs))))
	b.WriteString("\n")

	for i, pr := range m.PRs {
		if i < m.Cursor-m.Height || i > m.Cursor+m.Height {
			continue
		}

		stateMark := prStateMark(pr.State)
		timeAgo := formatTime(pr.CreatedAt)
		draft := ""
		if pr.IsDraft {
			draft = " [DRAFT]"
		}

		title := fmt.Sprintf("#%d %s", pr.Number, truncateStr(pr.Title, width-20))
		meta := fmt.Sprintf("  %s %s%s  %s  by %s",
			stateMark, title, draft, timeAgo, pr.Author)

		if i == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render(meta))
		} else {
			b.WriteString(styles.ListItemStyle.Render(meta))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// prStateMark returns a visual indicator for PR state.
func prStateMark(state models.PullRequestState) string {
	switch state {
	case models.PRStateOpen:
		return styles.StatusStagedStyle.Render("◆")
	case models.PRStateClosed:
		return styles.StatusDeletedStyle.Render("◆")
	case models.PRStateMerged:
		return styles.StatusUntrackedStyle.Render("◆")
	case models.PRStateDraft:
		return styles.StatusUnstagedStyle.Render("◇")
	default:
		return "◇"
	}
}
