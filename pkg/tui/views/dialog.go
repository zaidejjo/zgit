package views

import (
	"strings"

	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Modal sizing lives here so every dialog agrees on the budget: a box may
// never be wider than the terminal it is centred in. Getting this wrong is
// what clipped dialogs on narrow terminals and let a wide huh form burst
// out of its border.
//
// lipgloss geometry, for the boxes below (all of them are
// Border + Padding(0,1)):
//
//	total width = Style.Width + 2 (the border is drawn outside Width,
//	              while padding is counted inside it)
//	text area   = Style.Width - 2 (one cell of padding each side)
const (
	// boxBorder is the horizontal border (one cell each side).
	boxBorder = 2
	// boxPadding is the horizontal padding of a Padding(0,1) box.
	boxPadding = 2

	// dialogMaxContent caps content width so dialogs stay readable on
	// ultrawide terminals instead of stretching edge to edge.
	dialogMaxContent = 84
	// dialogMinContent keeps a dialog usable on a tiny terminal; below
	// this the frame guard clips rather than reflowing to nonsense.
	dialogMinContent = 24
	// dialogMargin keeps the box off the terminal edge.
	dialogMargin = 4
	// formFieldFrame is what a focused huh field draws outside its text:
	// a 1-cell left border plus 1 cell of padding.
	formFieldFrame = 2
)

// styleWidth returns the value to pass to Style.Width so a bordered,
// padded box occupies exactly total columns.
func styleWidth(total int) int {
	w := total - boxBorder
	if w < 1 {
		w = 1
	}
	return w
}

// contentWidth returns the text area available inside a bordered, padded
// box that occupies exactly total columns.
func contentWidth(total int) int {
	w := total - boxBorder - boxPadding
	if w < 1 {
		w = 1
	}
	return w
}

// dialogWidths returns, for a dialog rendered with styles.DialogBoxStyle
// at the given terminal width, the widest content block that keeps the
// box inside the terminal (borders and padding included), and the width
// to hand a huh form placed inside that block.
func dialogWidths(termWidth int) (content int, form int) {
	frame := styles.DialogBoxStyle.GetHorizontalFrameSize()
	content = termWidth - frame - dialogMargin
	if content > dialogMaxContent {
		content = dialogMaxContent
	}
	if content < dialogMinContent {
		content = dialogMinContent
	}
	form = content - formFieldFrame
	if form < 10 {
		form = 10
	}
	return content, form
}

// clampBlock hard-caps every line of a rendered block to width cells, so a
// dialog's content can never grow its border.
func clampBlock(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = components.Truncate(l, width)
	}
	return strings.Join(lines, "\n")
}
