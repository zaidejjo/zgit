package components

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// stripANSI removes SGR sequences so tests can assert on visible text.
func stripANSI(s string) string {
	return regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]").ReplaceAllString(s, "")
}

func testList() ZList[string] {
	l := ZList[string]{Height: 3}
	l.Items = []string{"dev", "feat/a", "feat/b", "main", "fix/c"}
	l.Match = func(s, q string) bool {
		return strings.Contains(strings.ToLower(s), q)
	}
	l.Render = func(s string, _ int) string { return " " + s }
	return l
}

func TestZListNav(t *testing.T) {
	l := testList()
	if !l.Nav("j") || l.Cursor != 1 {
		t.Fatalf("j = %d", l.Cursor)
	}
	l.Nav("G")
	if l.Cursor != 4 {
		t.Fatalf("G = %d", l.Cursor)
	}
	l.Nav("j") // clamped at end
	if l.Cursor != 4 {
		t.Fatalf("clamp = %d", l.Cursor)
	}
	l.Nav("g")
	l.Nav("pgdown")
	if l.Cursor != 3 {
		t.Fatalf("pgdown = %d", l.Cursor)
	}
	if l.Nav("x") {
		t.Fatalf("unknown key should not consume")
	}
	// Empty list never goes negative.
	e := ZList[string]{Height: 3}
	e.Nav("G")
	e.Nav("pgdown")
	if e.Cursor != 0 {
		t.Fatalf("empty cursor = %d", e.Cursor)
	}
}

func TestZListFilter(t *testing.T) {
	l := testList()
	l.Filtering = true
	for _, k := range []string{"f", "e", "a", "t"} {
		if !l.TypeKey(k) {
			t.Fatalf("typing %q not consumed", k)
		}
	}
	if len(l.Filtered()) != 2 {
		t.Fatalf("filtered = %v", l.Filtered())
	}
	l.TypeKey("backspace")
	if len(l.Filtered()) != 2 { // "fea" still matches feat/a + feat/b
		t.Fatalf("after backspace = %v", l.Filtered())
	}
	l.TypeKey("esc")
	if l.Filtering || l.Filter != "" {
		t.Fatalf("esc should clear + close")
	}
	l.Filtering = true
	l.TypeKey("z")
	l.TypeKey("enter")
	if l.Filtering || l.Filter != "z" {
		t.Fatalf("enter should keep filter")
	}
	if l.TypeKey("j") {
		t.Fatalf("idle list should not consume typing")
	}
}

func TestZListSelectAndRows(t *testing.T) {
	l := testList()
	sel, ok := l.Selected()
	if !ok || sel != "dev" {
		t.Fatalf("selected = %q %v", sel, ok)
	}
	l.Cursor = 99
	l.ClampCursor()
	if l.Cursor != 0 {
		t.Fatalf("clamp = %d", l.Cursor)
	}
	out := l.ViewRows(40)
	rows := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1
	if rows != 3 {
		t.Fatalf("windowed rows = %d, want 3", rows)
	}
	if !strings.Contains(out, "❯dev") {
		t.Fatalf("cursor row missing:\n%s", out)
	}
}

func TestToastsPushExpire(t *testing.T) {
	tt := NewToasts(time.Hour)
	if !tt.Empty() {
		t.Fatalf("fresh stack should be empty")
	}
	tt.Push("a", ToneInfo)
	tt.Push("b", ToneSuccess)
	tt.Push("c", ToneError)
	tt.Push("d", ToneInfo) // cap 3 drops "a"
	if got := tt.View(); strings.Contains(got, "\na\n") || strings.Contains(got, " a\n") {
		t.Fatalf("oldest should drop:\n%s", got)
	}
	if !strings.Contains(tt.View(), "d") {
		t.Fatalf("newest kept:\n%s", tt.View())
	}
	fast := NewToasts(20 * time.Millisecond)
	fast.Push("x", ToneInfo)
	time.Sleep(40 * time.Millisecond)
	if fast.Expire() {
		t.Fatalf("expired toast should clear")
	}
	if !fast.Empty() {
		t.Fatalf("should be empty")
	}
}

func TestTruncateWidth(t *testing.T) {
	styled := "\x1b[32m●\x1b[0m hello world, this is long"
	got := Truncate(styled, 10)
	if w := lipgloss.Width(got); w > 10 {
		t.Fatalf("width %d > 10: %q", w, got)
	}
	// Escapes must be complete (no half-written "\x1b[3" left behind).
	if dangling := regexp.MustCompile("\x1b\\[[0-9;]*$").FindString(got); dangling != "" {
		t.Fatalf("truncation split an escape: %q", got)
	}
	if plain := stripANSI(got); plain != "● hello w…" {
		t.Fatalf("plain = %q", plain)
	}
	if Truncate("", 10) != "" || Truncate("ab", 0) != "" {
		t.Fatalf("edge cases")
	}
	// Wide chars count double.
	wide := Truncate("日本語テストX", 6)
	if lipgloss.Width(wide) > 6 {
		t.Fatalf("wide overflow: %q", wide)
	}
}

func TestCursorLine(t *testing.T) {
	styled := "\x1b[38;2;166;227;161m◆\x1b[0m 3fec75a msg"
	got := CursorLine(styled)
	if plain := stripANSI(got); plain != "❯ 3fec75a msg" {
		t.Fatalf("plain = %q", plain)
	}
	// Styling is preserved (that is the point of the ANSI-aware swap).
	if !strings.Contains(got, "\x1b[38;2;166;227;161m") {
		t.Fatalf("color dropped: %q", got)
	}
	if CursorLine("") != "❯" || CursorLine("x") != "❯" {
		t.Fatalf("short lines need a bare cursor")
	}
	if plain := stripANSI(CursorLine("◆ abc")); plain != "❯ abc" {
		t.Fatalf("multibyte = %q", plain)
	}
}

func TestRowLineReservesStyleFrame(t *testing.T) {
	// A row style that pads must not push the row past the panel: the
	// padding is charged against the width, not added on top of it.
	styles.ListItemActiveStyle = lipgloss.NewStyle().PaddingLeft(1)
	defer styles.Refresh()
	for _, selected := range []bool{false, true} {
		got := RowLine("  ● some/long/path/file.go", 20, selected)
		if w := lipgloss.Width(got); w > 20 {
			t.Fatalf("selected=%v: row is %d cells, want <= 20 (%q)", selected, w, got)
		}
	}
}

// TestCursorLineWideFirstGrapheme covers the case that used to overflow by
// one cell: a wide (2-cell) first grapheme.
func TestCursorLineWideFirstGrapheme(t *testing.T) {
	got := CursorLine("日本語 abc")
	if w := lipgloss.Width(got); w > lipgloss.Width("日本語 abc") {
		t.Fatalf("cursor swap grew the line: %q (%d cells)", got, w)
	}
	if !strings.Contains(got, "abc") {
		t.Fatalf("tail lost: %q", got)
	}
}

// TestRowLineNeverExceedsWidth is the hard contract for every rendered row,
// including wide characters and a styled first grapheme.
func TestRowLineNeverExceedsWidth(t *testing.T) {
	cases := []struct {
		line     string
		width    int
		selected bool
	}{
		{"  ● services/platform/" + strings.Repeat("deep/", 20) + "file.go", 40, true},
		{"  ● services/platform/" + strings.Repeat("deep/", 20) + "file.go", 40, false},
		{"日本語のとても長いファイル名.go", 12, true},
		{strings.Repeat("🎉", 30), 20, true},
		{"", 10, true},
		{"x", 1, true},
		{"a", 0, false},
	}
	for _, c := range cases {
		got := RowLine(c.line, c.width, c.selected)
		if c.width <= 0 {
			continue
		}
		if w := lipgloss.Width(got); w > c.width {
			t.Fatalf("RowLine(%q, %d, %v) = %d cells: %q", c.line, c.width, c.selected, w, got)
		}
	}
}

func TestTruncateLeftKeepsTail(t *testing.T) {
	if got := TruncateLeft("short", 10); got != "short" {
		t.Fatalf("fits: %q", got)
	}
	if got := TruncateLeft("", 5); got != "" {
		t.Fatalf("empty: %q", got)
	}
	got := TruncateLeft("/home/someone/Projects/Apps/zgit", 12)
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "zgit") {
		t.Fatalf("tail lost: %q", got)
	}
	if w := lipgloss.Width(got); w > 12 {
		t.Fatalf("width %d > 12: %q", w, got)
	}
	// Multi-byte runes must survive whole. A 2-cell first grapheme is the
	// case ansi.TruncateLeft cannot cut into — asking it for one cell used
	// to return the string untouched, so a "shave until it fits" loop spun
	// forever. This test hanging is that bug.
	wide := TruncateLeft("日本語のとても長いパス/ファイル", 8)
	if w := lipgloss.Width(wide); w > 8 {
		t.Fatalf("wide width %d: %q", w, wide)
	}
	if strings.ContainsRune(wide, '\uFFFD') {
		t.Fatalf("split a rune: %q", wide)
	}
	// Every width from 1 to the full length must return in bounded time
	// and stay inside the budget.
	long := "日本語のとても長いパス/ファイル"
	for w := 1; w <= lipgloss.Width(long); w++ {
		got := TruncateLeft(long, w)
		if cw := lipgloss.Width(got); cw > w {
			t.Fatalf("width %d produced %d cells: %q", w, cw, got)
		}
		if strings.ContainsRune(got, '\uFFFD') {
			t.Fatalf("width %d split a rune: %q", w, got)
		}
	}
	// Zero-width budget is empty, not a lone ellipsis.
	if got := TruncateLeft("abcdef", 0); got != "" {
		t.Fatalf("zero width: %q", got)
	}
}

func TestCutLeftDropsWholeGraphemes(t *testing.T) {
	if got := CutLeft("abcdef", 0); got != "abcdef" {
		t.Fatalf("no cut: %q", got)
	}
	if got := CutLeft("abcdef", 2); got != "cdef" {
		t.Fatalf("ascii cut: %q", got)
	}
	if got := CutLeft("abc", 99); got != "" {
		t.Fatalf("over-cut: %q", got)
	}
	// A wide character straddling the cut is dropped whole: the result may
	// be narrower than requested, never wider.
	for cells := 1; cells <= 8; cells++ {
		got := CutLeft("日本語abc", cells)
		if w := lipgloss.Width(got); w > lipgloss.Width("日本語abc")-cells {
			t.Fatalf("cut %d kept %d cells: %q", cells, w, got)
		}
		if strings.ContainsRune(got, '\uFFFD') {
			t.Fatalf("cut %d split a rune: %q", cells, got)
		}
	}
	// Styling that belongs to the kept text survives the cut.
	styled := "\x1b[31mabcdef\x1b[0m"
	got := CutLeft(styled, 2)
	if plain := stripANSI(got); plain != "cdef" {
		t.Fatalf("styled cut = %q", plain)
	}
	if !strings.Contains(got, "\x1b[31m") {
		t.Fatalf("colour dropped: %q", got)
	}
}

// TestCursorLineKeepsLeadingEscapes pins the styled case: the escapes in
// front of the replaced grapheme must survive (the row keeps its colour),
// and the swap must never widen the line.
func TestCursorLineKeepsLeadingEscapes(t *testing.T) {
	const esc = "\x1b[38;2;166;227;161m"
	cases := []string{
		esc + "日本 abc",
		esc + "◆ 3fec75a msg",
		"日本語 abc",
	}
	for _, line := range cases {
		got := CursorLine(line)
		if w, max := lipgloss.Width(got), lipgloss.Width(line); w > max {
			t.Fatalf("CursorLine(%q) = %d cells > %d: %q", line, w, max, got)
		}
		if strings.HasPrefix(line, esc) && !strings.Contains(got, esc) {
			t.Fatalf("CursorLine(%q) dropped styling: %q", line, got)
		}
		if plain := stripANSI(got); !strings.HasPrefix(plain, "❯") {
			t.Fatalf("CursorLine(%q) = %q, want cursor first", line, plain)
		}
	}
	// A one-cell line is *entirely* the cursor, so the styling it carried
	// is gone with the replaced text — the cursor lands in the row style.
	if plain := stripANSI(CursorLine(esc + "x")); plain != "❯" {
		t.Fatalf("one-cell line = %q, want a bare cursor", plain)
	}
}
