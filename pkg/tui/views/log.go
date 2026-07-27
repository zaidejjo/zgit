package views

import (
	"fmt"
	"strings"
	"time"

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

		// Build tree prefix for this row
		treeStr := ""
		if treePrefix != nil && i < len(treePrefix) {
			treeStr = treePrefix[i]
		}

		// Compact: hash(7) + message (no author/date on separate line to save space)
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		// Available width after tree prefix
		treeWidth := len(treeStr)
		availWidth := width - treeWidth - 1
		if availWidth < 10 {
			availWidth = 10
		}

		msg := truncateStr(c.Message, availWidth-9)
		timeStr := formatTimeCompact(c.Timestamp)
		line := fmt.Sprintf("%s %s %s %s", treeStr, hash, msg, timeStr)

		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemSelectedStyle.Render(line))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
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
