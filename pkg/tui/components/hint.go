package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Hint is one key/description pair for the footer.
type Hint struct {
	Key  string
	Desc string
}

// Hints renders every hint, separated by a dim dot.
func Hints(hints []Hint) string {
	if len(hints) == 0 {
		return ""
	}
	g := glyphs.Current()
	sep := " " + styles.HintSepStyle.Render(g.Dash) + " "
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, hintString(h))
	}
	return strings.Join(parts, sep)
}

// HintCells is the width Hints occupies for the given pairs.
func HintCells(hints []Hint) int {
	if len(hints) == 0 {
		return 0
	}
	g := glyphs.Current()
	w := 0
	for i, h := range hints {
		if i > 0 {
			// The " · " separator between pairs.
			w += 2 + ansi.StringWidth(g.Dash)
		}
		w += ansi.StringWidth(h.Key)
		if h.Desc != "" {
			w += 1 + ansi.StringWidth(h.Desc)
		}
	}
	return w
}

// FitHints renders the longest suffix of hints that fits width, so the
// least important hint is the first to go. Order hints least to most
// important.
func FitHints(hints []Hint, width int) string {
	if width <= 0 || len(hints) == 0 {
		return ""
	}
	for start := 0; start < len(hints); start++ {
		tail := hints[start:]
		s := Hints(tail)
		if ansi.StringWidth(s) <= width {
			return s
		}
	}
	return ""
}

func hintString(h Hint) string {
	key := styles.HintKeyStyle.Render(h.Key)
	if h.Desc == "" {
		return key
	}
	return key + " " + styles.HintDescStyle.Render(h.Desc)
}
