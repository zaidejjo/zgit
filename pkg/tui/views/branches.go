package views

import (
	"fmt"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// BranchModel holds state for the branch list view.
type BranchModel struct {
	Branches []*models.Branch
	Cursor   int
	Offset   int
	Error    string
	Height   int
}

// NewBranchModel creates a default branch view model.
func NewBranchModel() BranchModel {
	return BranchModel{Height: 20}
}

// UpdateBranches refreshes the branch list.
func (m *BranchModel) UpdateBranches(branches []*models.Branch) {
	m.Branches = branches
	m.Error = ""
}

// View renders the branch list.
func (m BranchModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if len(m.Branches) == 0 {
		return styles.LoadingStyle.Render("No branches")
	}

	var b strings.Builder

	m.ensureCursorVisible()

	visible := m.Branches
	if len(visible) > m.Height {
		end := m.Offset + m.Height
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[m.Offset:end]
	}

	for i, br := range visible {
		globalIdx := m.Offset + i
		marker := "  "
		if br.IsHead {
			marker = styles.StatusStagedStyle.Render("*")
		}

		trackInfo := ""
		if br.Upstream != "" {
			track := ""
			if br.Ahead > 0 {
				track += fmt.Sprintf(" +%d", br.Ahead)
			}
			if br.Behind > 0 {
				track += fmt.Sprintf(" -%d", br.Behind)
			}
			trackInfo = styles.SubtitleStyle.Render(fmt.Sprintf(" [%s%s]", br.Upstream, track))
		}

		msg := ""
		if br.LatestMsg != "" {
			msg = styles.SubtitleStyle.Render(" " + truncateStr(br.LatestMsg, width-40))
		}

		line := fmt.Sprintf(" %s %s%s%s", marker, br.Name, trackInfo, msg)

		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render(line))
		} else if br.IsHead {
			b.WriteString(styles.ListItemStyle.Render(line))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *BranchModel) ensureCursorVisible() {
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
}
