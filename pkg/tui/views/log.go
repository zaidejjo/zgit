package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// LogModel holds state for the commit log view.
type LogModel struct {
	Commits    []*models.Commit
	Cursor     int
	Offset     int // scroll offset
	Error      string
	Height     int      // visible height
	treePrefix []string // cached tree graph strings
}

// NewLogModel creates a default log view model.
func NewLogModel() LogModel {
	return LogModel{Height: 20}
}

// UpdateLog refreshes the commit list and rebuilds tree graph.
func (m *LogModel) UpdateLog(commits []*models.Commit) {
	m.Commits = commits
	m.Error = ""
	m.treePrefix = RenderTreeGraph(commits)
}

// View renders the commit log with tree graph.
// Layout: [tree] [hash] [branch badges] [message]
func (m LogModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}

	if len(m.Commits) == 0 {
		return styles.LoadingStyle.Render("No commits yet")
	}

	var b strings.Builder

	// Adjust offset so cursor stays visible
	m.ensureCursorVisible()

	visible := m.Commits
	treePrefix := m.treePrefix
	if len(visible) > m.Height {
		end := m.Offset + m.Height
		if end > len(visible) {
			end = len(visible)
			m.Offset = end - m.Height
			if m.Offset < 0 {
				m.Offset = 0
			}
		}
		visible = visible[m.Offset:end]
		if treePrefix != nil && m.Offset < len(treePrefix) {
			treeEnd := m.Offset + len(visible)
			if treeEnd > len(treePrefix) {
				treeEnd = len(treePrefix)
			}
			treePrefix = treePrefix[m.Offset:treeEnd]
		} else {
			treePrefix = nil
		}
	}

	for i, c := range visible {
		globalIdx := m.Offset + i

		// Tree graph prefix
		treeStr := ""
		if treePrefix != nil && i < len(treePrefix) {
			treeStr = treePrefix[i]
		}

		line := commitRow(c, treeStr, width)
		b.WriteString(components.RowLine(line, width, globalIdx == m.Cursor))
		b.WriteString("\n")
	}

	return b.String()
}

// Commit rows are a fixed grid:
//
//	<graph> <hash> <age> <subject…> <refs>
//
// Only the subject is bright. The hash and the age are reference data you
// read once, refs answer "where am I", and the subject is what you scan
// for — the old layout bolded every ref and left the message the dimmest
// thing on the row, which is exactly backwards.
const (
	commitHashWidth = 7
	commitAgeWidth  = 4
	// commitMinSubject is the narrowest subject worth showing; below it
	// the row drops its ref chips rather than shredding the message.
	commitMinSubject = 16
)

// commitRow lays out one commit row, without the cursor affordance.
func commitRow(c *models.Commit, tree string, width int) string {
	hash := c.Hash
	if len(hash) > commitHashWidth {
		hash = hash[:commitHashWidth]
	}

	// A one-cell gutter keeps every row aligned and gives the cursor
	// somewhere to land that is not part of the content.
	const gutter = " "
	graphW := lipgloss.Width(tree)
	fixed := lipgloss.Width(gutter) + graphW + 1 + commitHashWidth + 1 + commitAgeWidth + 1

	age := ""
	if c.Timestamp.IsZero() {
		fixed -= commitAgeWidth + 1 // nothing to say: reclaim the column
	} else {
		age = formatTimeCompact(c.Timestamp)
	}

	refs := refChips(c.RefNames, refBudget(width-fixed))
	refsW := 0
	if refs != "" {
		refsW = lipgloss.Width(refs) + 1 // the space in front of the chips
	}

	subjectW := width - fixed - refsW
	if subjectW < commitMinSubject && refsW > 0 {
		refs, refsW, subjectW = "", 0, width-fixed
	}
	if subjectW < 1 {
		subjectW = 1
	}

	var b strings.Builder
	b.WriteString(gutter)
	b.WriteString(tree)
	b.WriteString(" ")
	b.WriteString(styles.LogHashStyle.Render(hash))
	b.WriteString(" ")
	if age != "" {
		b.WriteString(styles.MetaStyle.Render(padCells(age, commitAgeWidth)))
		b.WriteString(" ")
	}
	b.WriteString(padCells(truncateStr(c.Message, subjectW), subjectW))
	if refs != "" {
		b.WriteString(" ")
		b.WriteString(refs)
	}
	return b.String()
}

// padCells pads plain text with spaces to exactly width cells.
func padCells(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// refBudget is how many cells the ref column may take: at most half of
// the space left after the fixed columns, so the subject always keeps the
// other half.
func refBudget(remaining int) int {
	if remaining < 12 {
		return 0
	}
	return remaining / 2
}

// refChips renders a commit's decorations as whole chips within budget,
// so a branch and a tag are distinguishable at a glance and a chip is
// never cut in half: a half-drawn chip looks like a rendering bug.
func refChips(refNames string, budget int) string {
	if refNames == "" || budget < 6 {
		return ""
	}
	g := glyphs.Current()

	// Per-chip cap: one long branch name must not eat the whole column.
	cap := 26
	if budget < 2*cap {
		cap = budget - 2
	}

	var chips []string
	used := 0
	overflow := 0
	for _, part := range strings.Split(refNames, ", ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Skip the HEAD pointer itself.
		if part == "HEAD" {
			continue
		}
		// Take the right side after " -> ": the branch HEAD points to.
		if idx := strings.LastIndex(part, " -> "); idx >= 0 {
			part = part[idx+4:]
		}
		if part == "" {
			continue
		}

		var chip string
		switch {
		case strings.HasPrefix(part, "tag:"):
			tag := strings.TrimSpace(strings.TrimPrefix(part, "tag:"))
			chip = components.StateChip(components.Truncate(g.Tag+" "+tag, cap), styles.Peach)
		case strings.Contains(part, "/"):
			// A remote-tracking ref: real information, never the headline.
			chip = components.StateChip(components.Truncate(part, cap), styles.Subtext)
		default:
			chip = components.StateChip(components.Truncate(part, cap), styles.Teal)
		}

		w := lipgloss.Width(chip)
		if used+w > budget {
			// Count the rest so the row can say "+3" instead of hiding it.
			overflow++
			continue
		}
		if len(chips) > 0 {
			used++ // the space between chips
		}
		used += w
		chips = append(chips, chip)
	}

	if len(chips) == 0 {
		return ""
	}
	out := strings.Join(chips, " ")
	if overflow > 0 {
		more := styles.MetaStyle.Render(fmt.Sprintf("+%d", overflow))
		if used+1+lipgloss.Width(more) <= budget {
			out += " " + more
		}
	}
	return out
}

func (m *LogModel) ensureCursorVisible() {
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
}

func formatTimeCompact(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	}
}

// formatTime is used by all view files in this package.
func formatTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

// truncateStr shortens plain (unstyled) text to maxLen cells. Delegates to
// the shared ANSI/wide-char aware implementation so there is exactly one
// truncation rule in the TUI.
func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	return components.Truncate(s, maxLen)
}
