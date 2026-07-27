package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// StatusModel holds state for the working tree status view.
type StatusModel struct {
	Status        *models.Status
	Cursor        int
	ShowStaged    bool
	ShowUnstaged  bool
	ShowUntracked bool
	Error         string
}

// NewStatusModel creates a default status view model.
func NewStatusModel() StatusModel {
	return StatusModel{
		ShowStaged:    true,
		ShowUnstaged:  true,
		ShowUntracked: true,
	}
}

// UpdateStatus refreshes the status data.
func (m *StatusModel) UpdateStatus(s *models.Status) {
	m.Status = s
	m.Error = ""
}

// View renders the status screen.
func (m StatusModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if m.Status == nil {
		return styles.LoadingStyle.Render("Loading status...")
	}

	var b strings.Builder

	// Branch header
	b.WriteString(renderBranchHeader(m.Status))
	b.WriteString("\n")

	// Staged files
	if m.ShowStaged {
		staged := m.Status.StagedFiles()
		if len(staged) > 0 {
			b.WriteString(sectionTitle(fmt.Sprintf("Staged (%d)", len(staged))))
			b.WriteString("\n")
			for i, f := range staged {
				line := fmt.Sprintf("  %s  %s", statusMark("staged"), truncatePath(f.Path, width-6))
				if i == m.Cursor && m.ShowStaged {
					b.WriteString(styles.ListItemActiveStyle.Render(line))
				} else {
					b.WriteString(styles.ListItemStyle.Render(line))
				}
				b.WriteString("\n")
			}
		}
	}

	// Unstaged files
	if m.ShowUnstaged {
		unstaged := m.Status.UnstagedFiles()
		if len(unstaged) > 0 {
			b.WriteString(sectionTitle(fmt.Sprintf("Unstaged (%d)", len(unstaged))))
			b.WriteString("\n")
			for i, f := range unstaged {
				line := fmt.Sprintf("  %s  %s", statusMark("unstaged"), truncatePath(f.Path, width-6))
				if i == m.Cursor && !m.ShowStaged && m.ShowUnstaged {
					b.WriteString(styles.ListItemActiveStyle.Render(line))
				} else {
					b.WriteString(styles.ListItemStyle.Render(line))
				}
				b.WriteString("\n")
			}
		}
	}

	// Untracked files
	if m.ShowUntracked {
		untracked := m.Status.UntrackedFiles()
		if len(untracked) > 0 {
			b.WriteString(sectionTitle(fmt.Sprintf("Untracked (%d)", len(untracked))))
			b.WriteString("\n")
			for _, f := range untracked {
				line := fmt.Sprintf("  %s  %s", statusMark("untracked"), truncatePath(f.Path, width-6))
				b.WriteString(styles.ListItemStyle.Render(line))
				b.WriteString("\n")
			}
		}
	}

	if m.Status.IsClean {
		b.WriteString(styles.SubtitleStyle.Render("\n  Clean working tree\n"))
	}

	return b.String()
}

func renderBranchHeader(s *models.Status) string {
	parts := []string{styles.StatusBarBranchStyle.Render(s.Branch)}
	if s.Upstream != "" {
		track := ""
		if s.Ahead > 0 || s.Behind > 0 {
			track = fmt.Sprintf(" [")
			if s.Ahead > 0 {
				track += fmt.Sprintf("+%d", s.Ahead)
			}
			if s.Ahead > 0 && s.Behind > 0 {
				track += " "
			}
			if s.Behind > 0 {
				track += fmt.Sprintf("-%d", s.Behind)
			}
			track += "]"
		}
		parts = append(parts, styles.SubtitleStyle.Render(s.Upstream+track))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func sectionTitle(title string) string {
	return styles.SectionTitleStyle.Render(title)
}

func statusMark(kind string) string {
	switch kind {
	case "staged":
		return styles.StatusStagedStyle.Render("●")
	case "unstaged":
		return styles.StatusUnstagedStyle.Render("●")
	case "untracked":
		return styles.StatusUntrackedStyle.Render("○")
	case "deleted":
		return styles.StatusDeletedStyle.Render("✕")
	default:
		return " "
	}
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen < 3 {
		return path[:maxLen]
	}
	return "…" + path[len(path)-maxLen+1:]
}
