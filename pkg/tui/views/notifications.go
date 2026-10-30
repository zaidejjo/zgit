package views

import (
	"fmt"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// NotesModel holds state for the GitHub notifications inbox.
type NotesModel struct {
	components.ZList[*models.Notification]
	ShowAll bool // false = unread only (server-side)
	Loaded  bool
	Error   string
}

// NewNotesModel creates the notifications view model.
func NewNotesModel() NotesModel {
	m := NotesModel{}
	m.Height = 20
	m.Match = func(it *models.Notification, q string) bool {
		hay := strings.ToLower(strings.Join([]string{
			it.Title, it.RepoName, it.Reason, it.Type,
		}, " "))
		return strings.Contains(hay, q)
	}
	return m
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
	g := glyphs.Current()
	switch t {
	case "PullRequest":
		return styles.EmphasisStyle.Render(g.Merge)
	case "Issue":
		return styles.StatusStagedStyle.Render(g.Info)
	case "CheckSuite":
		return styles.StatusUnstagedStyle.Render(g.Check)
	case "Commit":
		return styles.HeaderBranchStyle.Render(g.Commit)
	case "Release":
		return styles.StatusUnstagedStyle.Render(g.Tag)
	default:
		return styles.SubtitleStyle.Render(g.Dot)
	}
}

// View renders the inbox, unread first.
func (m *NotesModel) View(width int) string {
	body := m.body(width)
	mode := "Unread"
	if m.ShowAll {
		mode = "All"
	}
	meta := mode + " " + screenMeta(len(m.Filtered()), len(m.Items), m.Filter)
	return screenChrome("Notifications", meta, body, width)
}

// body renders everything under the screen chrome.
func (m *NotesModel) body(width int) string {
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

	var b strings.Builder
	b.WriteString(m.FilterBar())

	m.EnsureVisible()
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
			dot = styles.EmphasisStyle.Render(glyphs.Current().Dot)
		}
		reason := it.Reason
		if reason != "" {
			reason = " [" + strings.ReplaceAll(reason, "_", " ") + "]"
		}
		line := fmt.Sprintf(" %s %s %s  %s%s",
			dot, noteTypeMark(it.Type), truncateStr(it.Title, 44),
			styles.SubtitleStyle.Render(truncateStr(it.RepoName, 26)), styles.SubtitleStyle.Render(reason))
		b.WriteString(components.RowLine(line, width, globalIdx == m.Cursor))
		b.WriteString("\n")
	}

	return b.String()
}
