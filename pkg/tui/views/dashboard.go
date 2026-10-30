package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/components"
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
	components.ZList[DashEntry]
	Loaded bool // false while the async scan runs
	Error  string
}

// Entries aliases the underlying items (read-only snapshot).
func (m *DashModel) Entries() []DashEntry {
	return m.Items
}

// NewDashModel creates a dashboard view model.
func NewDashModel() DashModel {
	m := DashModel{}
	m.Height = 20
	m.Match = func(e DashEntry, q string) bool {
		return strings.Contains(strings.ToLower(e.Name+" "+e.Path), q)
	}
	m.Render = dashRow
	return m
}

// UpdateEntries refreshes the dashboard data.
func (m *DashModel) UpdateEntries(entries []DashEntry) {
	m.Items = entries
	m.Loaded = true
	m.Error = ""
	m.ClampCursor()
}

// Selected returns the cursor entry (filter-aware).
func (m *DashModel) Selected() *DashEntry {
	e, ok := m.ZList.Selected()
	if !ok {
		return nil
	}
	out := e
	return &out
}

// View renders the dashboard list.
func (m *DashModel) View(width int) string {
	body := m.body(width)
	return screenChrome("Repositories", screenMeta(len(m.Filtered()), len(m.Items), m.Filter), body, width)
}

// body renders everything under the screen chrome.
func (m *DashModel) body(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if !m.Loaded {
		return styles.LoadingStyle.Render("Scanning repositories...")
	}
	if len(m.Items) == 0 {
		return styles.SubtitleStyle.Render("No repositories yet — open one with 'zgit -C <path>'")
	}

	var b strings.Builder
	b.WriteString(m.FilterBar())
	if len(m.Filtered()) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No repositories match filter")
	}

	return b.String() + m.ViewRows(width)
}

// dashRow renders one repository row (no cursor — ZList wraps it).
func dashRow(e DashEntry, width int) string {
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
		openTag = styles.EmphasisStyle.Render(" ‹open›")
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

	return line
}
