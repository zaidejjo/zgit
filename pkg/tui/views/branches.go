package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
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
//
// Columns, not concatenation: the branch name owns a fixed column, the
// tracking state is a chip, and the last commit message fills the rest.
// The old row glued name and message together ("chore/bench-interleave
// chore(bench): interleave…"), which reads as one long word.
func branchRow(br *models.Branch, width int) string {
	g := glyphs.Current()

	const gutter = " "
	marker := " "
	nameStyle := styles.ListItemStyle
	if br.IsHead {
		marker = styles.EmphasisStyle.Render(g.Head)
		nameStyle = lipgloss.NewStyle().Foreground(styles.TrackAdd).Bold(true)
	}

	nameW := width / 3
	if nameW < 12 {
		nameW = 12
	}
	if nameW > 30 {
		nameW = 30
	}
	name := components.Truncate(br.Name, nameW)

	// Tracking: the upstream name only when it is not just the branch
	// itself, plus the divergence arrows.
	track := ""
	if br.Upstream != "" && br.Upstream != "origin/"+br.Name {
		track = br.Upstream
	}
	if arrows := trackingArrow(br.Ahead, br.Behind); arrows != "" {
		if track != "" {
			track += " "
		}
		track += arrows
	}
	chip := ""
	chipW := 0
	if track != "" {
		chip = components.StateChip(track, styles.Blue)
		chipW = lipgloss.Width(chip) + 1
	}

	// gutter(1) + marker(1) + space(1) + name column + space(1)
	fixed := 4 + nameW
	subjectW := width - fixed - chipW
	if subjectW < 8 {
		// The message is the first thing to go: it is supporting detail.
		chip, chipW, subjectW = "", 0, width-fixed
	}
	if subjectW < 1 {
		// Even the name column is too wide for the pane: shrink it, so
		// the gutter and the marker stay aligned with every other row.
		nameW += subjectW - 1
		if nameW < 3 {
			nameW = 3
		}
		name = components.Truncate(br.Name, nameW)
		subjectW = 0
	}

	var b strings.Builder
	b.WriteString(gutter)
	b.WriteString(marker)
	b.WriteString(" ")
	b.WriteString(padCells(nameStyle.Render(name), nameW))
	b.WriteString(" ")
	if chip != "" {
		b.WriteString(chip)
		b.WriteString(" ")
	}
	if br.LatestMsg != "" && subjectW > 1 {
		b.WriteString(styles.SubtextStyle.Render(components.Truncate(br.LatestMsg, subjectW)))
	}
	return components.Truncate(b.String(), width)
}
