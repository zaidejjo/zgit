package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// TestOverlayCenterIsColumnAccurate pins the compositor: the modal must
// land on the same cell column whatever the background contains, and the
// composited rows must still fill the terminal exactly. Rune-index math
// used to shift the modal whenever the background had wide characters.
func TestOverlayCenterIsColumnAccurate(t *testing.T) {
	const width = 40
	const rows = 7

	modal := "╭──╮\n│ab│\n╰──╯"
	modalW := 4
	wantCol := (width - modalW) / 2

	bases := []string{
		"",
		"ascii background",
		"日本語のテキストです",
		"🎉 emoji background",
		"mixed 日本 and 🎉 and ascii",
	}

	for _, base := range bases {
		visible := ansi.Truncate(base, width, "")
		bg := visible + strings.Repeat(" ", width-ansi.StringWidth(visible))
		lines := make([]string, rows)
		for i := range lines {
			lines[i] = bg
		}

		out := overlayCenter(strings.Join(lines, "\n"), modal, width, rows)
		got := strings.Split(out, "\n")
		if len(got) != rows {
			t.Fatalf("base %q: %d rows, want %d", base, len(got), rows)
		}

		col, row := -1, -1
		for i, l := range got {
			if w := lipgloss.Width(l); w != width {
				t.Fatalf("base %q: row %d is %d cells, want %d: %q", base, i, w, width, l)
			}
			plain := ansi.Strip(l)
			if idx := strings.Index(plain, "╭──╮"); idx >= 0 {
				col, row = ansi.StringWidth(plain[:idx]), i
			}
		}
		if row < 0 {
			t.Fatalf("base %q: modal missing from composite:\n%s", base, out)
		}
		if col != wantCol {
			t.Fatalf("base %q: modal starts at column %d, want %d", base, col, wantCol)
		}
	}
}
