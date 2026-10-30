package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// TestPaneOccupiesExactFrame is the container contract: every row is
// exactly Width cells, and the pane is exactly Height rows.
func TestPaneOccupiesExactFrame(t *testing.T) {
	body := strings.Repeat("a very long body line that will not fit ", 6) + "\n" +
		"日本語のとても長い行" + "\n" + "short"
	for _, width := range []int{12, 30, 80, 137} {
		for _, height := range []int{0, 3, 10, 40} {
			p := NewPane("Log", width, true)
			p.Meta = "12 commits"
			p.Height = height
			p.Body = body
			lines := strings.Split(p.View(), "\n")
			if height > 0 && len(lines) != height {
				t.Fatalf("width=%d height=%d: rendered %d rows", width, height, len(lines))
			}
			for i, line := range lines {
				if w := ansi.StringWidth(line); w != width {
					t.Fatalf("width=%d height=%d: row %d is %d cells: %q", width, height, i, w, line)
				}
			}
		}
	}
}

// TestPaneKeepsTitleWhenNarrow: the name is the last thing to go.
func TestPaneKeepsTitleWhenNarrow(t *testing.T) {
	p := NewPane("Branches", 12, false)
	p.Meta = "128 branches"
	out := p.View()
	if !strings.Contains(out, "BRANCHES") {
		t.Fatalf("title lost at width 12:\n%s", out)
	}
	if strings.Contains(out, "128 branches") {
		t.Fatalf("meta should be dropped before the title:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(line); w != 12 {
			t.Fatalf("row is %d cells: %q", w, line)
		}
	}
}

// TestPaneRuleGrowsWithWidth keeps the chrome glued to the pane.
func TestPaneRuleGrowsWithWidth(t *testing.T) {
	g := glyphs.Current()
	for _, width := range []int{10, 40, 120} {
		p := NewPane("Status", width, true)
		rows := strings.Split(p.View(), "\n")
		if len(rows) < 2 {
			t.Fatalf("width=%d: no chrome rows", width)
		}
		if got := ansi.StringWidth(rows[1]); got != width {
			t.Fatalf("width=%d: rule row is %d cells", width, got)
		}
		if g.Rule != "" && strings.Count(rows[1], g.Rule) != width {
			t.Fatalf("width=%d: rule is not continuous", width)
		}
	}
}

// TestPaneEmptyBody checks the chrome-only case (a pane with nothing to
// show yet).
func TestPaneEmptyBody(t *testing.T) {
	p := NewPane("Log", 20, false)
	p.Height = 4
	rows := strings.Split(p.View(), "\n")
	if len(rows) != 4 {
		t.Fatalf("rows = %d", len(rows))
	}
	for i, r := range rows {
		if w := ansi.StringWidth(r); w != 20 {
			t.Fatalf("row %d is %d cells", i, w)
		}
	}
}

func TestPaneZeroWidthIsEmpty(t *testing.T) {
	if got := (Pane{}).View(); got != "" {
		t.Fatalf("zero width rendered %q", got)
	}
}

// TestChipWidthIsTextPlusPadding: callers budget with ChipCells, so it
// must agree with what Chip renders.
func TestChipWidthIsTextPlusPadding(t *testing.T) {
	for _, text := range []string{"dev", "feature/long-branch-name", "↑2 ↓1"} {
		got := MutedChip(text)
		if w, want := ansi.StringWidth(got), ChipCells(text); w != want {
			t.Fatalf("Chip(%q) is %d cells, ChipCells says %d", text, w, want)
		}
	}
	if MutedChip("") != "" || ChipCells("") != 0 {
		t.Fatal("empty chip must render nothing and cost nothing")
	}
}

// TestSegmentWidthIsTextPlusPadding covers the separator arithmetic in
// every glyph mode: the Powerline mode spends one extra cell.
func TestSegmentWidthIsTextPlusPadding(t *testing.T) {
	defer glyphs.Use(glyphs.Unicode)
	for _, mode := range []glyphs.Mode{glyphs.Unicode, glyphs.ASCII, glyphs.Nerd} {
		glyphs.Use(mode)
		for _, text := range []string{"dev", "/home/someone/Projects/zgit"} {
			got := Segment(text, styles.Text, styles.ChipBg)
			if w, want := ansi.StringWidth(got), SegmentCells(text); w != want {
				t.Fatalf("mode=%v Segment(%q) is %d cells, SegmentCells says %d", mode, text, w, want)
			}
		}
	}
}

// TestFitHintsDropsFromTheFront: hints are ordered least to most
// important, so the tail is what survives.
func TestFitHintsDropsFromTheFront(t *testing.T) {
	hints := []Hint{
		{Key: "ctrl+p", Desc: "palette"},
		{Key: "L", Desc: "theme"},
		{Key: "?", Desc: "help"},
	}
	for _, width := range []int{1, 6, 8, 12, 20, 40, 200} {
		got := FitHints(hints, width)
		if w := ansi.StringWidth(got); w > width {
			t.Fatalf("width=%d: got %d cells: %q", width, w, got)
		}
		if width >= HintCells(hints) && got != Hints(hints) {
			t.Fatalf("width=%d: full set fits but was trimmed: %q", width, got)
		}
		if got != "" && !strings.Contains(got, "help") {
			t.Fatalf("width=%d: most important hint dropped: %q", width, got)
		}
	}
	if got := FitHints(nil, 80); got != "" {
		t.Fatalf("no hints = %q", got)
	}
}

// TestHintCellsAgreesWithRendering keeps the budget honest.
func TestHintCellsAgreesWithRendering(t *testing.T) {
	hints := []Hint{{Key: "?", Desc: "help"}, {Key: "q", Desc: "quit"}}
	if w, want := ansi.StringWidth(Hints(hints)), HintCells(hints); w != want {
		t.Fatalf("Hints is %d cells, HintCells says %d: %q", w, want, Hints(hints))
	}
	// A key-only hint still measures right.
	keys := []Hint{{Key: "esc"}, {Key: "enter"}}
	if w, want := ansi.StringWidth(Hints(keys)), HintCells(keys); w != want {
		t.Fatalf("key-only hints: %d != %d", w, want)
	}
}
