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
	Commits []*models.Commit
	Cursor  int
	Offset  int // scroll offset
	Error   string
	Height  int // visible height
}

// NewLogModel creates a default log view model.
func NewLogModel() LogModel {
	return LogModel{Height: 20}
}

// UpdateLog refreshes the commit list.
func (m *LogModel) UpdateLog(commits []*models.Commit) {
	m.Commits = commits
	m.Error = ""
}

// View renders the commit log.
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
	}

	for i, c := range visible {
		globalIdx := m.Offset + i
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render(
				fmt.Sprintf(" %s %s", hash, truncateStr(c.Message, width-20)),
			))
			b.WriteString("\n")
			b.WriteString(styles.ListItemActiveStyle.Render(
				fmt.Sprintf("   %s  %s", c.Author, formatTime(c.Timestamp)),
			))
		} else {
			b.WriteString(styles.ListItemStyle.Render(
				fmt.Sprintf(" %s %s", hash, truncateStr(c.Message, width-10)),
			))
			b.WriteString("\n")
			b.WriteString(styles.SubtitleStyle.Render(
				fmt.Sprintf("   %s  %s", c.Author, formatTime(c.Timestamp)),
			))
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
