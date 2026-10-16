package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// BranchModel holds state for the branch list view.
type BranchModel struct {
	components.ZList[*models.Branch]
	Error string
}

// NewBranchModel creates a default branch view model.
func NewBranchModel() BranchModel {
	m := BranchModel{}
	m.Height = 20
	m.Match = func(br *models.Branch, q string) bool {
		hay := strings.ToLower(strings.Join([]string{br.Name, br.Upstream, br.LatestMsg}, " "))
		return strings.Contains(hay, q)
	}
	m.Render = branchRow
	return m
}

// UpdateBranches refreshes the branch list.
func (m *BranchModel) UpdateBranches(branches []*models.Branch) {
	m.Items = branches
	m.Error = ""
	m.ClampCursor()
}

// View renders the branch list.
func (m *BranchModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if len(m.Items) == 0 {
		return styles.LoadingStyle.Render("No branches")
	}

	return m.ViewRows(width)
}

// branchRow renders one branch row (no cursor affordance — ZList wraps it).
func branchRow(br *models.Branch, width int) string {
	marker := "  "
	name := br.Name
	if br.IsHead {
		marker = styles.StatusStagedStyle.Render("*")
		name = styles.StatusStagedStyle.Render(br.Name)
	}

	// Self-tracking upstreams (origin/<name>) repeat the branch
	// name — show only ahead/behind counts for those.
	trackInfo := ""
	if br.Upstream != "" {
		up := br.Upstream
		self := up == "origin/"+br.Name
		if self {
			up = ""
		}
		track := ""
		if br.Ahead > 0 {
			track += fmt.Sprintf("↑%d", br.Ahead)
		}
		if br.Behind > 0 {
			if track != "" {
				track += " "
			}
			track += fmt.Sprintf("↓%d", br.Behind)
		}
		if up != "" {
			if track != "" {
				track = " " + track
			}
			trackInfo = styles.SubtitleStyle.Render(fmt.Sprintf(" [%s%s]", up, track))
		} else if track != "" {
			trackInfo = " " + styles.SubtitleStyle.Render(track)
		}
	}

	// Calculate available width for message: marker (2) + " " + name + trackInfo + " " + msg
	markerWidth := lipgloss.Width(marker)
	trackWidth := lipgloss.Width(trackInfo)
	nameWidth := lipgloss.Width(name)
	msgMax := width - markerWidth - 1 - nameWidth - 1 - trackWidth - 1
	if msgMax < 0 {
		msgMax = 0
	}

	msg := ""
	if br.LatestMsg != "" && msgMax > 0 {
		trimmed := truncateStr(br.LatestMsg, msgMax)
		msg = styles.SubtitleStyle.Render(" " + trimmed)
	}

	line := fmt.Sprintf(" %s %s%s%s", marker, name, trackInfo, msg)

	// Verify line fits panel width — if not, strip msg entirely
	if lipgloss.Width(line) > width {
		msg = ""
		line = fmt.Sprintf(" %s %s%s", marker, name, trackInfo)
		// If still too wide, truncate name
		if lipgloss.Width(line) > width {
			availName := width - markerWidth - 1 - trackWidth - 1
			if availName < 1 {
				availName = 1
			}
			truncName := truncateStr(br.Name, availName)
			line = fmt.Sprintf(" %s %s%s", marker, truncName, trackInfo)
		}
	}

	return line
}
