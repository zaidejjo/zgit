package components

import (
	"strings"
	"testing"
)

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

func TestCursorLine(t *testing.T) {
	if got := CursorLine("\x1b[31m●\x1b[0m hi"); got != "❯ hi" {
		t.Fatalf("= %q", got)
	}
	if CursorLine("") != "❯" {
		t.Fatalf("empty")
	}
}
