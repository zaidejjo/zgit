package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Truncate hard-caps a rendered line to width cells, ANSI- and
// wide-char aware. Final guardrail: no row may ever wrap the panel.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

// CursorLine retargets the first visible grapheme of a rendered line to ❯
// without touching the rest: ANSI- and grapheme-aware, so escapes and wide
// characters survive (never slice styled strings by byte/rune — that
// decapitates escapes into visible "[38;2;…" garbage).
//
// The whole first grapheme is replaced, never a single cell: cutting one
// cell out of a 2-cell character would leave it in place and *grow* the
// row by a cell, which wraps the panel.
func CursorLine(line string) string {
	if ansi.StringWidth(line) <= 1 {
		return "❯"
	}

	// Walk to the first grapheme that occupies cells. Leading escape
	// sequences carry none, but they set the colour of the text after
	// them, so they are kept in front of the cursor.
	var (
		state   byte
		offset  int
		leading strings.Builder
	)
	cw := 0
	for offset < len(line) {
		unit, w, n, newState := ansi.DecodeSequence(line[offset:], state, nil)
		if n <= 0 {
			break
		}
		state = newState
		offset += n
		if w == 0 {
			leading.WriteString(unit)
			continue
		}
		cw = w
		break
	}
	if cw <= 0 {
		return "❯"
	}
	if cw >= ansi.StringWidth(line) {
		// The first grapheme is the whole line: the cursor alone fits.
		return leading.String() + "❯"
	}
	return leading.String() + "❯" + line[offset:]
}

// CutLeft drops the first cells cells of a rendered line and returns the
// rest, keeping escape sequences and never splitting a grapheme: a wide
// character straddling the cut is dropped whole, so the result can be one
// cell narrower than asked for, never wider.
//
// ansi.TruncateLeft cannot express this. It refuses to cut *into* a
// grapheme, so asking it to drop one cell in front of a 2-cell character
// hands the string straight back — a caller looping until the tail fits
// would never finish.
func CutLeft(s string, cells int) string {
	if cells <= 0 {
		return s
	}
	var (
		state   byte
		offset  int
		dropped int
		leading strings.Builder
	)
	for offset < len(s) && dropped < cells {
		unit, w, n, newState := ansi.DecodeSequence(s[offset:], state, nil)
		if n <= 0 {
			break
		}
		state = newState
		if w == 0 {
			// Zero-width units cost no cells, so they ride along with the
			// tail and keep its styling.
			leading.WriteString(unit)
			offset += n
			continue
		}
		offset += n
		dropped += w
	}
	if offset >= len(s) {
		return ""
	}
	return leading.String() + s[offset:]
}

// TruncateLeft keeps the tail of a line and prefixes an ellipsis, so the
// end of a long path (the part that identifies it) stays on screen.
func TruncateLeft(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w <= width {
		return s
	}
	// One cell of the budget goes to the ellipsis.
	return "…" + CutLeft(s, w-width+1)
}

// RowLine renders one list row as exactly width cells. It reserves the
// frame its style adds (padding/margin/border), hard-caps the text to
// what is left, then draws the cursor when selected. Every row goes
// through here, so a row can never grow wider than its panel and make
// the terminal wrap the whole layout.
func RowLine(line string, width int, selected bool) string {
	st := styles.ListItemStyle
	if selected {
		st = styles.ListItemActiveStyle
	}
	inner := width - st.GetHorizontalFrameSize()
	if inner < 1 {
		inner = 1
	}
	body := Truncate(line, inner)
	if selected {
		// A wide first grapheme can make the cursor swap return one cell
		// more than it consumed, so cap again: exactly is the contract.
		body = Truncate(CursorLine(body), inner)
	}
	return st.Render(body)
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
		b.WriteString(RowLine(line, width, l.Offset+i == l.Cursor))
		b.WriteString("\n")
	}
	return b.String()
}
