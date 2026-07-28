package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
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
		treeWidth := lipgloss.Width(treeStr)

		// Short hash (7 chars)
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		// Available width for message: width - tree - space - hash(7) - space
		msgMax := width - treeWidth - 1 - 7 - 1
		if msgMax < 1 {
			msgMax = 1
		}

		// Parse branch/tag refs for badges
		var badgeStr string
		if c.RefNames != "" {
			badgeStr = renderRefBadges(c.RefNames)
		}
		badgeWidth := lipgloss.Width(badgeStr)
		if badgeStr != "" {
			badgeStr = " " + badgeStr
			badgeWidth++ // leading space
		}

		// Adjust msg for badge space
		msgMaxAdjusted := msgMax - badgeWidth
		if msgMaxAdjusted < 0 {
			msgMaxAdjusted = 0
		}
		msg := truncateStr(c.Message, msgMaxAdjusted)

		// Build line: treeStr + " " + hash + badge + " " + msg
		line := fmt.Sprintf("%s %s%s %s", treeStr, hash, badgeStr, msg)
		lineWidth := lipgloss.Width(line)
		if lineWidth > width {
			// Overflow — trim msg further
			overflow := lineWidth - width
			msg = truncateStr(c.Message, msgMaxAdjusted-overflow)
			if msgMaxAdjusted-overflow < 0 {
				msg = ""
			}
			line = fmt.Sprintf("%s %s%s %s", treeStr, hash, badgeStr, msg)
		}

		if globalIdx == m.Cursor {
			// Cursor: ❯ prefix replaces first rune (handles multi-byte tree symbols)
			runes := []rune(line)
			if len(runes) > 0 {
				cursorLine := "❯" + string(runes[1:])
				b.WriteString(styles.ListItemActiveStyle.Render(cursorLine))
			} else {
				b.WriteString(styles.ListItemActiveStyle.Render("❯"))
			}
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderRefBadges parses RefNames (e.g. "HEAD -> main, origin/main")
// and returns compact colored badges like "[main] [origin/main]".
// Filters out HEAD and remote tracking refs for brevity.
func renderRefBadges(refNames string) string {
	if refNames == "" {
		return ""
	}
	var badges []string
	for _, part := range strings.Split(refNames, ", ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Skip HEAD pointer
		if part == "HEAD" {
			continue
		}
		// Take the right side after " -> " (branch that HEAD points to)
		if idx := strings.LastIndex(part, " -> "); idx >= 0 {
			part = part[idx+4:]
		}
		if part == "" {
			continue
		}
		// Style the badge
		color := styles.Teal
		if strings.HasPrefix(part, "origin/") {
			color = styles.Subtext
			part = part[len("origin/"):]
		}
		badge := lipgloss.NewStyle().
			Foreground(color).
			Bold(true).
			Render(part)
		badges = append(badges, badge)
	}
	return strings.Join(badges, " ")
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

func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
