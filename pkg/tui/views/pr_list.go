package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// PRListModel holds state for the pull request list view.
type PRListModel struct {
	PRs    []*models.PullRequestSummary
	Cursor int
	Offset int
	Error  string
	Height int
}

// NewPRListModel creates a default PR list view model.
func NewPRListModel() PRListModel {
	return PRListModel{Height: 20}
}

// UpdatePRs refreshes the PR list data.
func (m *PRListModel) UpdatePRs(prs []*models.PullRequestSummary) {
	m.PRs = prs
	m.Error = ""
	if m.Cursor >= len(prs) {
		m.Cursor = 0
	}
}

// SelectedPR returns the currently selected PR summary.
func (m *PRListModel) SelectedPR() *models.PullRequestSummary {
	if len(m.PRs) == 0 || m.Cursor < 0 || m.Cursor >= len(m.PRs) {
		return nil
	}
	return m.PRs[m.Cursor]
}

// View renders the PR list with state badges and branch info.
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

	m.ensureCursorVisible()

	var b strings.Builder

	b.WriteString(styles.SectionTitleStyle.Render(fmt.Sprintf(" Pull Requests (%d)", len(m.PRs))))
	b.WriteString("\n")

	visible := m.PRs
	if len(visible) > m.Height {
		end := m.Offset + m.Height
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[m.Offset:end]
	}

	for i, pr := range visible {
		globalIdx := m.Offset + i

		// State mark (colored diamond)
		stateMark := prStateMark(pr.State)

		// Draft badge
		draftBadge := ""
		if pr.IsDraft {
			draftBadge = styles.StatusUnstagedStyle.Render(" DRAFT ")
		}

		// Branch info: head → base
		branchInfo := fmt.Sprintf("%s → %s", pr.HeadRef, pr.BaseRef)

		// Mergeable status
		mergeableStr := ""
		switch pr.Mergeable {
		case "MERGEABLE":
			mergeableStr = styles.StatusStagedStyle.Render(" ✔")
		case "CONFLICTING":
			mergeableStr = styles.ErrorStyle.Render(" ✗")
		}

		// Review state
		reviewStr := ""
		switch pr.ReviewState {
		case "APPROVED":
			reviewStr = styles.StatusStagedStyle.Render(" ✓")
		case "CHANGES_REQUESTED":
			reviewStr = styles.ErrorStyle.Render(" !")
		case "REVIEW_REQUIRED":
			reviewStr = styles.SubtitleStyle.Render(" ?")
		}

		// Author tag
		authorTag := ""
		if pr.Author != "" {
			authorTag = styles.SubtitleStyle.Render(" @" + pr.Author)
		}

		// Format line: stateMark #N title badges | @author head→base review mergeable
		header := fmt.Sprintf("#%d %s", pr.Number, truncateStr(pr.Title, 60))
		line := fmt.Sprintf(" %s %s%s%s  %s%s%s",
			stateMark, header, draftBadge, authorTag, branchInfo, reviewStr, mergeableStr)

		// Verify line fits — if too long, truncate title further
		if lipgloss.Width(line) > width {
			availTitle := width - lipgloss.Width(fmt.Sprintf(" %s #%d %s  %s%s",
				stateMark, pr.Number, authorTag, branchInfo, mergeableStr)) - 1
			if availTitle < 3 {
				availTitle = 3
			}
			header = fmt.Sprintf("#%d %s", pr.Number, truncateStr(pr.Title, availTitle))
			line = fmt.Sprintf(" %s %s%s%s  %s%s%s",
				stateMark, header, draftBadge, authorTag, branchInfo, reviewStr, mergeableStr)
		}

		if globalIdx == m.Cursor {
			// ❯ replaces leading space
			b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *PRListModel) ensureCursorVisible() {
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
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
