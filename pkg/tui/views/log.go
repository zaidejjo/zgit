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

	refs := refChips(c.RefNames)
	refsW := 0
	if refs != "" {
		refsW = lipgloss.Width(refs)
		if budget := (width - fixed) / 2; refsW > budget {
			if budget < 6 {
				refs, refsW = "", 0
			} else {
				refs = components.Truncate(refs, budget)
				refsW = lipgloss.Width(refs)
			}
		}
		if refs != "" {
			refsW++ // the space in front of the chips
		}
	}

	subjectW := width - fixed - refsW
	if subjectW < commitMinSubject && refsW > 0 {
		refs, subjectW = "", width-fixed
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

// refChips renders a commit's decorations as one chip per ref, so a
// branch and a tag are distinguishable at a glance: local branches take
// the branch hue, tags take the tag hue, remotes stay quiet.
func refChips(refNames string) string {
	if refNames == "" {
		return ""
	}
	g := glyphs.Current()
	var chips []string
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
		switch {
		case strings.HasPrefix(part, "tag: "):
			tag := strings.TrimPrefix(part, "tag: ")
			chips = append(chips, components.StateChip(strings.TrimSpace(g.Tag+" "+tag), styles.Peach))
		case strings.Contains(part, "/"):
			// Remote-tracking ref: real information, but never the
			// headline. Dim, and labelled with its remote.
			chips = append(chips, components.StateChip(part, styles.Subtext))
		default:
			chips = append(chips, components.StateChip(part, styles.Teal))
		}
	}
	if len(chips) == 0 {
		return ""
	}
	// Three chips are as much as a row can carry before the refs become
	// louder than the commit itself.
	const maxChips = 3
	if len(chips) > maxChips {
		extra := len(chips) - maxChips
		chips = append(chips[:maxChips], styles.MetaStyle.Render(fmt.Sprintf("+%d", extra)))
	}
	return strings.Join(chips, " ")
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
