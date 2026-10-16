package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// NotesModel holds state for the GitHub notifications inbox.
type NotesModel struct {
	Items     []*models.Notification
	ShowAll   bool // false = unread only (server-side)
	Loaded    bool
	Cursor    int
	Offset    int
	Error     string
	Height    int
	Filter    string
	Filtering bool
}

// NewNotesModel creates the notifications view model.
func NewNotesModel() NotesModel {
	return NotesModel{Height: 20}
}

// UpdateNotes refreshes the inbox.
func (m *NotesModel) UpdateNotes(items []*models.Notification) {
	m.Items = items
	m.Loaded = true
	m.Error = ""
	m.ClampCursor()
}

// UnreadCount counts unread items locally (for the tab badge).
func (m *NotesModel) UnreadCount() int {
	n := 0
	for _, it := range m.Items {
		if it.Unread {
			n++
		}
	}
	return n
}

// MarkRead flips one thread locally (server confirm happens in app keys).
func (m *NotesModel) MarkRead(id string) {
	for _, it := range m.Items {
		if it.ID == id {
			it.Unread = false
		}
	}
}

// MarkAllRead flips everything locally.
func (m *NotesModel) MarkAllRead() {
	for _, it := range m.Items {
		it.Unread = false
	}
}

// Filtered returns items matching the filter (title/repo/reason).
func (m *NotesModel) Filtered() []*models.Notification {
	q := strings.ToLower(strings.TrimSpace(m.Filter))
	if q == "" {
		return m.Items
	}
	out := make([]*models.Notification, 0, len(m.Items))
	for _, it := range m.Items {
		hay := strings.ToLower(strings.Join([]string{
			it.Title, it.RepoName, it.Reason, it.Type,
		}, " "))
		if strings.Contains(hay, q) {
			out = append(out, it)
		}
	}
	return out
}

// ClampCursor keeps the cursor inside the filtered list.
func (m *NotesModel) ClampCursor() {
	if m.Cursor >= len(m.Filtered()) {
		m.Cursor = 0
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

// Ordered returns filtered items unread-first (stable within groups).
// Both View and Selected use it so the cursor always matches.
func (m *NotesModel) Ordered() []*models.Notification {
	visible := m.Filtered()
	ordered := make([]*models.Notification, 0, len(visible))
	for _, it := range visible {
		if it.Unread {
			ordered = append(ordered, it)
		}
	}
	for _, it := range visible {
		if !it.Unread {
			ordered = append(ordered, it)
		}
	}
	return ordered
}

// Selected returns the cursor item (filter-aware).
func (m *NotesModel) Selected() *models.Notification {
	visible := m.Ordered()
	if len(visible) == 0 || m.Cursor < 0 || m.Cursor >= len(visible) {
		return nil
	}
	return visible[m.Cursor]
}

func noteTypeMark(t string) string {
	switch t {
	case "PullRequest":
		return styles.ListItemActiveStyle.Render("◈")
	case "Issue":
		return styles.StatusStagedStyle.Render("●")
	case "CheckSuite":
		return styles.StatusUnstagedStyle.Render("◉")
	case "Commit":
		return styles.HeaderBranchStyle.Render("⬢")
	case "Release":
		return styles.StatusUnstagedStyle.Render("✦")
	default:
		return styles.SubtitleStyle.Render("○")
	}
}

// View renders the inbox, unread first.
func (m *NotesModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if !m.Loaded {
		return styles.LoadingStyle.Render("Loading notifications...")
	}
	if len(m.Items) == 0 {
		if m.ShowAll {
			return styles.SubtitleStyle.Render("No notifications at all — enjoy the quiet")
		}
		return styles.SubtitleStyle.Render("Inbox zero — nothing unread")
	}

	m.ensureCursorVisible()

	var b strings.Builder
	mode := "Unread"
	if m.ShowAll {
		mode = "All"
	}
	title := fmt.Sprintf(" Notifications [%s] (%d)", mode, len(m.Items))
	if strings.TrimSpace(m.Filter) != "" {
		title = fmt.Sprintf(" Notifications (%d/%d) filter: %s", len(m.Filtered()), len(m.Items), m.Filter)
	}
	b.WriteString(styles.SectionTitleStyle.Render(title))
	b.WriteString("\n")
	if m.Filtering {
		b.WriteString(styles.FilterBarStyle.Render(" /" + m.Filter + "█"))
		b.WriteString("\n")
	}

	visible := m.Ordered()

	if len(visible) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No notifications match filter")
	}
	if len(visible) > m.Height {
		end := m.Offset + m.Height
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[m.Offset:end]
	}

	for i, it := range visible {
		globalIdx := m.Offset + i

		dot := " "
		if it.Unread {
			dot = styles.ListItemActiveStyle.Render("●")
		}
		reason := it.Reason
		if reason != "" {
			reason = " [" + strings.ReplaceAll(reason, "_", " ") + "]"
		}
		line := fmt.Sprintf(" %s %s %s  %s%s",
			dot, noteTypeMark(it.Type), truncateStr(it.Title, 44),
			styles.SubtitleStyle.Render(truncateStr(it.RepoName, 26)), styles.SubtitleStyle.Render(reason))
		if lipgloss.Width(line) > width {
			line = fmt.Sprintf(" %s %s %s", dot, noteTypeMark(it.Type), truncateStr(it.Title, width-8))
		}

		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *NotesModel) ensureCursorVisible() {
	m.ClampCursor()
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
}
