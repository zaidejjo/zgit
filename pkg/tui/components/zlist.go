package components

import (
	"regexp"
	"strings"

	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// ansiEscape matches SGR/control sequences for width-safe reuse.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// CursorLine rebuilds a rendered line for the selected row: ANSI is
// stripped first (never slice styled strings by byte/rune — that
// decapitates escapes into visible "[38;2;…" garbage), the first
// visible rune becomes ❯, and the caller wraps it in the active style.
func CursorLine(line string) string {
	plain := ansiEscape.ReplaceAllString(line, "")
	runes := []rune(plain)
	if len(runes) == 0 {
		return "❯"
	}
	return "❯" + string(runes[1:])
}

// ZList is the single filterable, scrollable list for every screen.
// Item rendering stays with the view (domain formatting); cursor,
// filter, paging and selection live here exactly once.
//
// Match maps an item to its searchable text; Render maps an item to a
// styled line WITHOUT any cursor affordance (selection wraps it).
type ZList[T any] struct {
	Items     []T
	Cursor    int
	Offset    int
	Height    int
	Filter    string
	Filtering bool
	Match     func(T, string) bool
	Render    func(T, int) string
}

// Filtered returns items matching the filter (case-insensitive).
// A nil Match means no filtering.
func (l *ZList[T]) Filtered() []T {
	q := strings.ToLower(strings.TrimSpace(l.Filter))
	if q == "" || l.Match == nil {
		return l.Items
	}
	out := make([]T, 0, len(l.Items))
	for _, it := range l.Items {
		if l.Match(it, q) {
			out = append(out, it)
		}
	}
	return out
}

// Selected returns the cursor item (filter-aware).
func (l *ZList[T]) Selected() (T, bool) {
	visible := l.Filtered()
	var zero T
	if len(visible) == 0 || l.Cursor < 0 || l.Cursor >= len(visible) {
		return zero, false
	}
	return visible[l.Cursor], true
}

// ClampCursor keeps the cursor inside the filtered list.
func (l *ZList[T]) ClampCursor() {
	if l.Cursor >= len(l.Filtered()) {
		l.Cursor = 0
	}
	if l.Cursor < 0 {
		l.Cursor = 0
	}
}

// EnsureVisible scrolls the window to the cursor.
func (l *ZList[T]) EnsureVisible() {
	l.ClampCursor()
	if l.Cursor < l.Offset {
		l.Offset = l.Cursor
	}
	if l.Cursor >= l.Offset+l.Height {
		l.Offset = l.Cursor - l.Height + 1
	}
}

// Nav handles j/k/g/G/home/end/pgup/pgdn. True = key consumed.
func (l *ZList[T]) Nav(key string) bool {
	n := len(l.Filtered())
	switch key {
	case "j", "down":
		if n > 0 && l.Cursor < n-1 {
			l.Cursor++
		}
		return true
	case "k", "up":
		if l.Cursor > 0 {
			l.Cursor--
		}
		return true
	case "g", "home":
		l.Cursor = 0
		return true
	case "G", "end":
		l.Cursor = n - 1
		if l.Cursor < 0 {
			l.Cursor = 0
		}
		return true
	case "pgdown", "ctrl+f":
		if n == 0 {
			return true
		}
		l.Cursor += l.Height
		if l.Cursor >= n {
			l.Cursor = n - 1
		}
		return true
	case "pgup", "ctrl+b":
		l.Cursor -= l.Height
		if l.Cursor < 0 {
			l.Cursor = 0
		}
		return true
	}
	return false
}

// TypeKey routes keystrokes to the filter input while Filtering.
// enter keeps the filter, esc clears it. True = consumed.
func (l *ZList[T]) TypeKey(key string) bool {
	if !l.Filtering {
		return false
	}
	switch key {
	case "esc", "enter":
		l.Filtering = false
		if key == "esc" {
			l.Filter = ""
		}
		l.ClampCursor()
	case "backspace":
		if len(l.Filter) > 0 {
			l.Filter = l.Filter[:len(l.Filter)-1]
			l.ClampCursor()
		}
	default:
		if len(key) == 1 || key == " " {
			l.Filter += key
			l.ClampCursor()
		}
	}
	return true
}

// FilterBar renders the active filter prompt, or "" when idle.
func (l *ZList[T]) FilterBar() string {
	if !l.Filtering {
		return ""
	}
	return styles.FilterBarStyle.Render(" /"+l.Filter+"█") + "\n"
}

// ViewRows renders the visible window with cursor selection.
// Render must be set; a nil Render yields blank rows.
func (l *ZList[T]) ViewRows(width int) string {
	l.EnsureVisible()
	visible := l.Filtered()
	if len(visible) > l.Height {
		end := l.Offset + l.Height
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[l.Offset:end]
	}
	var b strings.Builder
	for i, item := range visible {
		line := ""
		if l.Render != nil {
			line = l.Render(item, width)
		}
		if l.Offset+i == l.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render(CursorLine(line)))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}
