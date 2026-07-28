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
				// Replace first space with ❯ prefix. First byte is always ASCII space (safe byte slice).
				b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
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
				b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
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
				b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
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

// formatFilePath renders path as "filename.ext (dir/path/)" fitting maxLen.
// Priority order:
//  1. filename (dir/)       — full format
//  2. filename (…/right/)   — dir truncated with "…/" prefix + trailing /
//  3. filename              — dir dropped entirely
//  4. filename…             — filename truncated (only if terminal is too narrow)
func formatFilePath(path string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// Find last slash -> filename / dir
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}

	// No dir — just filename
	if lastSlash < 0 {
		if lipgloss.Width(path) <= maxLen {
			return path
		}
		// Filename alone too wide — truncate rune-aware
		runes := []rune(path)
		if maxLen-1 > 0 {
			return string(runes[:maxLen-1]) + "…"
		}
		return string(runes[:maxLen])
	}

	filename := path[lastSlash+1:]
	fnWidth := lipgloss.Width(filename)
	dir := path[:lastSlash]

	// Helper: truncate dir with "…/" prefix + trailing / inside (parens).
	// budget = total chars available for truncated dir including overhead.
	// Overhead: "…/" = 2, trailing "/" = 1 → 3 chars.
	// keep = budget - 3 = chars from right of dir to preserve.
	truncateDir := func(budget int) string {
		if budget < 4 { // "…/X/" minimum: 2+1+1 = 4
			return ""
		}
		dirRunes := []rune(dir)
		keep := budget - 3 // -3 for "…/" prefix + trailing /
		if keep < 1 {
			return ""
		}
		if len(dirRunes) <= keep {
			return "…/" + dir + "/"
		}
		return "…/" + string(dirRunes[len(dirRunes)-keep:]) + "/"
	}

	// Priority 1: full format "filename (dir/)"
	// overhead: " (" + "/)" = 2 + 2 = 4
	if fnWidth+4+lipgloss.Width(dir) <= maxLen {
		return fmt.Sprintf("%s (%s/)", filename, dir)
	}
	// Priority 1b: "filename (dir)" — try without trailing slash
	// overhead: " (" + ")" = 2 + 1 = 3
	if fnWidth+3+lipgloss.Width(dir) <= maxLen {
		return fmt.Sprintf("%s (%s)", filename, dir)
	}

	// Priority 2: "filename (…/right/)"
	// 3 chars overhead: " (" + ")" = space + parens
	// truncated dir format: "…/X/" = 2 (…/) + keep + 1 (/) = 3 + keep
	dirBudget := maxLen - fnWidth - 3
	if dirBudget >= 4 {
		truncDir := truncateDir(dirBudget)
		if truncDir != "" {
			formatted := fmt.Sprintf("%s (%s)", filename, truncDir)
			if lipgloss.Width(formatted) <= maxLen {
				return formatted
			}
		}
	}

	// Priority 3: filename only
	if fnWidth <= maxLen {
		return filename
	}

	// Priority 4: truncated filename — only when terminal is too narrow
	runes := []rune(filename)
	if maxLen-1 > 0 {
		return string(runes[:maxLen-1]) + "…"
	}
	return string(runes[:maxLen])
}
