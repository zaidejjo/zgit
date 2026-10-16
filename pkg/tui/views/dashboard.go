package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// DashEntry is one repository row on the dashboard.
type DashEntry struct {
	Name     string
	Path     string
	Branch   string
	Clean    bool
	Dirty    int
	Ahead    int
	Behind   int
	Favorite bool
	Current  bool
	Err      string
}

// DashModel holds state for the repository dashboard view.
type DashModel struct {
	Entries   []DashEntry
	Loaded    bool // false while the async scan runs
	Cursor    int
	Offset    int
	Error     string
	Height    int
	Filter    string
	Filtering bool
}

// NewDashModel creates a dashboard view model.
func NewDashModel() DashModel {
	return DashModel{Height: 20}
}

// UpdateEntries refreshes the dashboard data.
func (m *DashModel) UpdateEntries(entries []DashEntry) {
	m.Entries = entries
	m.Loaded = true
	m.Error = ""
	m.ClampCursor()
}

// Filtered returns entries matching the filter (name or path substring).
func (m *DashModel) Filtered() []DashEntry {
	q := strings.ToLower(strings.TrimSpace(m.Filter))
	if q == "" {
		return m.Entries
	}
	out := make([]DashEntry, 0, len(m.Entries))
	for _, e := range m.Entries {
		if strings.Contains(strings.ToLower(e.Name+" "+e.Path), q) {
			out = append(out, e)
		}
	}
	return out
}

// ClampCursor keeps the cursor inside the filtered list.
func (m *DashModel) ClampCursor() {
	if m.Cursor >= len(m.Filtered()) {
		m.Cursor = 0
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

// Selected returns the cursor entry (filter-aware).
func (m *DashModel) Selected() *DashEntry {
	visible := m.Filtered()
	if len(visible) == 0 || m.Cursor < 0 || m.Cursor >= len(visible) {
		return nil
	}
	return &visible[m.Cursor]
}

// View renders the dashboard list.
func (m *DashModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if !m.Loaded {
		return styles.LoadingStyle.Render("Scanning repositories...")
	}
	if len(m.Entries) == 0 {
		return styles.SubtitleStyle.Render("No repositories yet — open one with 'zgit -C <path>'")
	}

	m.ensureCursorVisible()

	var b strings.Builder
	title := fmt.Sprintf(" Repositories (%d)", len(m.Entries))
	if strings.TrimSpace(m.Filter) != "" {
		title = fmt.Sprintf(" Repositories (%d/%d) filter: %s", len(m.Filtered()), len(m.Entries), m.Filter)
	}
	b.WriteString(styles.SectionTitleStyle.Render(title))
	b.WriteString("\n")
	if m.Filtering {
		b.WriteString(styles.FilterBarStyle.Render(" /" + m.Filter + "█"))
		b.WriteString("\n")
	}

	visible := m.Filtered()
	if len(visible) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No repositories match filter")
	}
	if len(visible) > m.Height {
		end := m.Offset + m.Height
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[m.Offset:end]
	}

	for i, e := range visible {
		globalIdx := m.Offset + i

		var mark string
		switch {
		case e.Err != "":
			mark = styles.StatusDeletedStyle.Render("✗")
		case !e.Clean:
			mark = styles.StatusUnstagedStyle.Render("●")
		default:
			mark = styles.StatusStagedStyle.Render("●")
		}

		name := e.Name
		star := ""
		if e.Favorite {
			star = styles.StatusUnstagedStyle.Render("★ ")
		}
		openTag := ""
		if e.Current {
			openTag = styles.ListItemActiveStyle.Render(" ‹open›")
		}

		detail := e.Branch
		if e.Err != "" {
			detail = "unreachable"
		} else {
			if e.Dirty > 0 {
				detail += fmt.Sprintf(" %d dirty", e.Dirty)
			}
			if e.Ahead > 0 || e.Behind > 0 {
				detail += fmt.Sprintf(" +%d/-%d", e.Ahead, e.Behind)
			}
		}
		if detail == "" {
			detail = "—"
		}

		line := fmt.Sprintf(" %s %s%s%s  %s", mark, star, truncateStr(name, 38),
			openTag, styles.SubtitleStyle.Render(truncateStr(detail, 40)))
		if lipgloss.Width(line) > width {
			line = fmt.Sprintf(" %s %s%s", mark, star, truncateStr(name, width-8))
		}

		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render(cursorLine(line)))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *DashModel) ensureCursorVisible() {
	m.ClampCursor()
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
}
