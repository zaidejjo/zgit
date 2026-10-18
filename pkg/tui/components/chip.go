package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Chip draws an inline badge: " text " on a filled background.
//
// Pills are how the cockpit states a fact without spending a column on a
// label — a branch, a count, a review state. Every chip is exactly its
// text plus two cells, so callers can budget for it (ChipCells).
func Chip(text string, fg, bg lipgloss.Color) string {
	if text == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Render(" " + text + " ")
}

// ChipCells is the width Chip occupies for the given text.
func ChipCells(text string) int {
	if text == "" {
		return 0
	}
	return len([]rune(text)) + 2
}

// MutedChip states a fact at rest: dim text on the chip surface.
func MutedChip(text string) string {
	return Chip(text, styles.Text, styles.ChipBg)
}

// AccentChip is the one high-emphasis chip in a row — the current
// branch, the focused subject. Use sparingly: two of them compete.
func AccentChip(text string) string {
	return Chip(text, styles.ChipFgAccent, styles.ChipBgAccent)
}

// StateChip carries a semantic hue on the neutral chip surface: +3
// staged in green, −2 behind in red.
func StateChip(text string, fg lipgloss.Color) string {
	return Chip(text, fg, styles.ChipBg)
}

// Segment draws a status-bar block and its Powerline separator, so
// adjacent segments read as one continuous bar in a patched font. The
// separator inherits the segment's background and is one cell wide; in
// every other glyph mode the block is a plain padded chip.
func Segment(text string, fg, bg lipgloss.Color) string {
	g := glyphs.Current()
	block := lipgloss.NewStyle().Background(bg).Foreground(fg).Render(" " + text + " ")
	if g.PillRight == "" {
		return block
	}
	return block + lipgloss.NewStyle().Foreground(bg).Render(g.PillRight)
}

// SegmentCells is the width Segment occupies for the given text.
func SegmentCells(text string) int {
	g := glyphs.Current()
	w := len([]rune(text)) + 2
	if g.PillRight != "" {
		w++
	}
	return w
}
