package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// PRListModel holds state for the pull request list view.
type PRListModel struct {
	components.ZList[*models.PullRequestSummary]
	Error string
}

// NewPRListModel creates a default PR list view model.
func NewPRListModel() PRListModel {
	m := PRListModel{}
	m.Height = 20
	m.Match = func(pr *models.PullRequestSummary, q string) bool {
		hay := strings.ToLower(strings.Join([]string{
			fmt.Sprintf("%d", pr.Number), pr.Title, pr.Author, pr.HeadRef,
		}, " "))
		return strings.Contains(hay, q)
	}
	m.Render = prRow
	return m
}

// UpdatePRs refreshes the PR list data.
func (m *PRListModel) UpdatePRs(prs []*models.PullRequestSummary) {
	m.Items = prs
	m.Error = ""
	m.ClampCursor()
}

// SelectedPR returns the currently selected PR summary (filter-aware).
func (m *PRListModel) SelectedPR() *models.PullRequestSummary {
	pr, _ := m.Selected()
	return pr
}

// View renders the PR list with state badges and branch info.
func (m *PRListModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if m.Items == nil {
		return styles.LoadingStyle.Render("Loading pull requests...")
	}

	if len(m.Items) == 0 {
		return styles.SubtitleStyle.Render("No pull requests found")
	}

	var b strings.Builder

	title := fmt.Sprintf(" Pull Requests (%d)", len(m.Items))
	if strings.TrimSpace(m.Filter) != "" {
		title = fmt.Sprintf(" Pull Requests (%d/%d) filter: %s", len(m.Filtered()), len(m.Items), m.Filter)
	}
	b.WriteString(styles.SectionTitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(m.FilterBar())
	filtered := m.Filtered()
	if len(filtered) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No pull requests match filter")
	}

	return b.String() + m.ViewRows(width)
}

// prRow renders one PR row (no cursor affordance — ZList wraps it).
func prRow(pr *models.PullRequestSummary, width int) string {

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

	return line
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
