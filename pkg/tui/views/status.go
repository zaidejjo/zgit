package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
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

	b.WriteString(branchLine(m.Status, width))
	b.WriteString("\n")

	// Files are a table, not a paragraph: a fixed code column, a bright
	// filename, and the directory right-aligned so the eye can scan the
	// paths instead of re-anchoring on every row.
	section := func(label string, tone lipgloss.Color, files []models.FileStatus, offset int, code func(models.FileStatus) string) {
		if len(files) == 0 {
			return
		}
		b.WriteString(sectionLabel(label, len(files), tone, width))
		b.WriteString("\n")
		for i, f := range files {
			line := fileRow(f, code(f), width)
			b.WriteString(components.RowLine(line, width, offset+i == m.Cursor))
			b.WriteString("\n")
		}
	}

	if m.ShowStaged {
		section("Staged", styles.TrackAdd, staged, stagedOffset, func(f models.FileStatus) string {
			return f.Staged.String()
		})
	}
	if m.ShowUnstaged {
		section("Unstaged", styles.Warning, unstaged, unstagedOffset, func(f models.FileStatus) string {
			return f.Unstaged.String()
		})
	}
	if m.ShowUntracked {
		section("Untracked", styles.Subtext, untracked, untrackedOffset, func(f models.FileStatus) string {
			return "?"
		})
	}

	if m.Status.IsClean {
		b.WriteString(cleanState(width))
	}

	return b.String()
}

// branchLine states where you are: branch, divergence, upstream. The old
// header concatenated the branch and the upstream with no separator
// ("devorigin/dev"); chips cannot collide like that.
func branchLine(s *models.Status, width int) string {
	g := glyphs.Current()
	parts := []string{components.AccentChip(s.Branch)}

	if track := trackingArrow(s.Ahead, s.Behind); track != "" {
		parts = append(parts, components.StateChip(track, styles.Blue))
	}
	if s.Upstream != "" {
		up := s.Upstream
		if up == "origin/"+s.Branch {
			// Self-tracking: the arrows already said it.
			up = ""
		}
		if up != "" {
			parts = append(parts, styles.MetaStyle.Render(g.Right+" "+up))
		}
	}
	if s.IsMerging {
		parts = append(parts, components.StateChip("merging", styles.Warning))
	}
	if s.IsRebasing {
		parts = append(parts, components.StateChip("rebasing", styles.Warning))
	}
	if s.IsCherryPick {
		parts = append(parts, components.StateChip("cherry-pick", styles.Warning))
	}
	if s.IsReverting {
		parts = append(parts, components.StateChip("reverting", styles.Warning))
	}
	if s.IsBisecting {
		parts = append(parts, components.StateChip("bisecting", styles.Warning))
	}

	line := " " + strings.Join(parts, " ")
	if lipgloss.Width(line) > width {
		// Not even the chips fit: fall back to the branch alone, and
		// truncate it rather than overrunning the pane.
		line = " " + components.StateChip(components.Truncate(s.Branch, max(1, width-3)), styles.Teal)
	}
	return components.Truncate(line, width)
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// sectionLabel is the section heading: a quiet name and a loud count, so
// the numbers are the first thing a glance picks up.
func sectionLabel(label string, n int, tone lipgloss.Color, width int) string {
	g := glyphs.Current()
	mark := styles.PaneBarStyle.Render(g.Bar)
	text := styles.SectionLabelStyle.Render(strings.ToUpper(label))
	count := lipgloss.NewStyle().Foreground(tone).Bold(true).Render(fmt.Sprintf("%d", n))
	line := fmt.Sprintf(" %s %s %s", mark, text, count)
	if lipgloss.Width(line) > width {
		line = fmt.Sprintf(" %s %d", text, n)
	}
	return components.Truncate(line, width)
}

// fileRow lays out one working-tree row:
//
//	<code> <filename> … <directory/>
//
// The code column is the git status letter, not a decorative dot: M for
// modified, A for added, D for deleted, R for renamed.
func fileRow(f models.FileStatus, code string, width int) string {
	if code == "" {
		code = " "
	}
	code = string([]rune(code)[0])
	codeCell := lipgloss.NewStyle().Foreground(statusCodeColor(code)).Bold(true).Render(padCells(code, 1))

	path := f.Path
	if f.OldPath != "" {
		path = f.OldPath + " → " + f.Path
	}
	name, dir := splitPath(path)

	// gutter(1) + code(1) + space(1)
	const fixed = 3
	nameW := width - fixed
	dirW := 0
	if dir != "" && width >= 48 {
		dirW = lipgloss.Width(dir)
		if max := (width - fixed) / 3; dirW > max {
			dirW = max
		}
		dir = components.TruncateLeft(dir, dirW)
		dirW = lipgloss.Width(dir)
		nameW = width - fixed - dirW - 1
	}
	if nameW < 1 {
		nameW = 1
	}

	line := " " + codeCell + " " + padCells(styles.StatusFileNameStyle.Render(components.Truncate(name, nameW)), nameW)
	if dirW > 0 {
		line += " " + styles.MetaStyle.Render(padCells(dir, dirW))
	}
	return line
}

// splitPath separates a path into its file name and its directory.
func splitPath(path string) (name, dir string) {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:], path[:i+1]
	}
	return path, ""
}

// statusCodeColor maps a porcelain status letter to its hue.
func statusCodeColor(code string) lipgloss.Color {
	switch code {
	case "M":
		return styles.Warning
	case "A":
		return styles.TrackAdd
	case "D":
		return styles.TrackDel
	case "R":
		return styles.Info
	case "C":
		return styles.Teal
	case "U":
		return styles.Danger
	default:
		return styles.Overlay0
	}
}

// cleanState is the empty state: a clean tree is good news, so it is
// stated as a result rather than a top-left line of grey text with three
// hundred pixels of dead space under it.
func cleanState(width int) string {
	g := glyphs.Current()
	msg := lipgloss.NewStyle().Foreground(styles.TrackAdd).Bold(true).Render(g.Check + " working tree clean")
	hint := styles.MetaStyle.Render("nothing to commit")
	return centered(msg, width) + "\n" + centered(hint, width) + "\n"
}

// centered pads a rendered string with spaces so it sits in the middle of
// width cells.
func centered(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return components.Truncate(s, width)
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s
}

// trackingArrow renders a branch's divergence as "↑2↓1", or "" when the
// branch is level with its upstream.
func trackingArrow(ahead, behind int) string {
	g := glyphs.Current()
	switch {
	case ahead > 0 && behind > 0:
		return fmt.Sprintf("%s%d %s%d", g.Up, ahead, g.Down, behind)
	case ahead > 0:
		return fmt.Sprintf("%s%d", g.Up, ahead)
	case behind > 0:
		return fmt.Sprintf("%s%d", g.Down, behind)
	}
	return ""
}
