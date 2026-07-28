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
	Cursor        int // index into VisibleFiles
	ShowStaged    bool
	ShowUnstaged  bool
	ShowUntracked bool
	Error         string
}

// VisibleFiles returns the flat list of files currently visible in the panel,
// ordered as: staged, unstaged, untracked.
func (m *StatusModel) VisibleFiles() []models.FileStatus {
	if m.Status == nil {
		return nil
	}
	var out []models.FileStatus
	if m.ShowStaged {
		out = append(out, m.Status.StagedFiles()...)
	}
	if m.ShowUnstaged {
		out = append(out, m.Status.UnstagedFiles()...)
	}
	if m.ShowUntracked {
		out = append(out, m.Status.UntrackedFiles()...)
	}
	return out
}

// CursorFile returns the FileStatus at the current cursor position, or nil.
func (m *StatusModel) CursorFile() *models.FileStatus {
	visible := m.VisibleFiles()
	if m.Cursor < 0 || m.Cursor >= len(visible) {
		return nil
	}
	return &visible[m.Cursor]
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

// View renders the status screen with section-aware cursor indexing.
// Cursor is a flat index into VisibleFiles() (staged + unstaged + untracked).
// Each section computes its local cursor from the flat index.
func (m StatusModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if m.Status == nil {
		return styles.LoadingStyle.Render("Loading status...")
	}

	// Pre-compute file lists and section offsets
	staged := m.Status.StagedFiles()
	unstaged := m.Status.UnstagedFiles()
	untracked := m.Status.UntrackedFiles()

	stagedOffset := 0
	unstagedOffset := len(staged)
	untrackedOffset := unstagedOffset + len(unstaged)

	var b strings.Builder

	// Branch header
	b.WriteString(renderBranchHeader(m.Status))
	b.WriteString("\n")

	// Staged files
	if m.ShowStaged && len(staged) > 0 {
		b.WriteString(sectionTitle(fmt.Sprintf("Staged (%d)", len(staged))))
		b.WriteString("\n")
		for i, f := range staged {
			line := fmt.Sprintf("  %s  %s", statusMark("staged"), formatFilePath(f.Path, width-6))
			localIdx := stagedOffset + i
			if localIdx == m.Cursor {
				b.WriteString(styles.ListItemActiveStyle.Render(line))
			} else {
				b.WriteString(styles.ListItemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Unstaged files
	if m.ShowUnstaged && len(unstaged) > 0 {
		b.WriteString(sectionTitle(fmt.Sprintf("Unstaged (%d)", len(unstaged))))
		b.WriteString("\n")
		for i, f := range unstaged {
			line := fmt.Sprintf("  %s  %s", statusMark("unstaged"), formatFilePath(f.Path, width-6))
			localIdx := unstagedOffset + i
			if localIdx == m.Cursor {
				b.WriteString(styles.ListItemActiveStyle.Render(line))
			} else {
				b.WriteString(styles.ListItemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Untracked files
	if m.ShowUntracked && len(untracked) > 0 {
		b.WriteString(sectionTitle(fmt.Sprintf("Untracked (%d)", len(untracked))))
		b.WriteString("\n")
		for i, f := range untracked {
			line := fmt.Sprintf("  %s  %s", statusMark("untracked"), formatFilePath(f.Path, width-6))
			localIdx := untrackedOffset + i
			if localIdx == m.Cursor {
				b.WriteString(styles.ListItemActiveStyle.Render(line))
			} else {
				b.WriteString(styles.ListItemStyle.Render(line))
			}
			b.WriteString("\n")
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

// formatFilePath renders a path as "filename (dir/)" fitting within maxLen.
// Always keeps the filename fully visible. Truncates dir with "…" when needed.
// Falls back to filname-only if dir won't fit.
func formatFilePath(path string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// Find last slash to separate dir and filename
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash < 0 {
		// Just a filename — no dir
		if lipgloss.Width(path) <= maxLen {
			return path
		}
		// Truncate rune-aware to avoid cutting multi-byte
		runes := []rune(path)
		return string(runes[:maxLen-2]) + "…"
	}

	filename := path[lastSlash+1:]
	dir := path[:lastSlash]

	// Format: "filename (dir/)"
	// Try full format first
	formatted := fmt.Sprintf("%s (%s/)", filename, dir)
	if lipgloss.Width(formatted) <= maxLen {
		return formatted
	}

	// Try without trailing slash
	formatted = fmt.Sprintf("%s (%s)", filename, dir)
	if lipgloss.Width(formatted) <= maxLen {
		return formatted
	}

	// Try with truncated dir
	// Budget for: filename + " (" + dir + ")"
	// 4 chars overhead: " (./)" minimum
	dirBudget := maxLen - lipgloss.Width(filename) - 4
	if dirBudget < 1 {
		// No room for dir at all — show just filename
		if lipgloss.Width(filename) <= maxLen {
			return filename
		}
		// Truncate filename rune-aware, keep recognizable start
		fnRunes := []rune(filename)
		if maxLen-2 > 0 {
			return string(fnRunes[:maxLen-2]) + "…"
		}
		return string(fnRunes[:maxLen])
	}

	if lipgloss.Width(dir) > dirBudget {
		// Truncate dir from left, keep rightmost part
		dirRunes := []rune(dir)
		if dirBudget < 3 {
			// Not enough for "…" + dir — show just filename
			if lipgloss.Width(filename) <= maxLen {
				return filename
			}
			fnRunes := []rune(filename)
			if maxLen-2 > 0 {
				return string(fnRunes[:maxLen-2]) + "…"
			}
			return string(fnRunes[:maxLen])
		}
		// Truncate: "…" + rightmost dirBudget-1 chars
		truncDir := "…" + string(dirRunes[len(dirRunes)-(dirBudget-1):])
		formatted = fmt.Sprintf("%s (%s)", filename, truncDir)
		if lipgloss.Width(formatted) <= maxLen {
			return formatted
		}
	}

	// Final fallback: show truncated filename only
	if lipgloss.Width(filename) <= maxLen {
		return filename
	}
	fnRunes := []rune(filename)
	if maxLen-2 > 0 {
		return string(fnRunes[:maxLen-2]) + "…"
	}
	return string(fnRunes[:maxLen])
}
