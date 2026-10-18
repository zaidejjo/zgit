package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Pane is the cockpit container: a title row carrying an accent bar, a
// hairline rule, then the body.
//
// It replaces the bordered panel. A box costs four columns and two rows
// per pane and encodes nothing the rule does not: focus is already said
// by the bar and the rule colour, and the columns buy back ~8 columns of
// content on a 1360-pixel terminal.
type Pane struct {
	Title   string // pane name, drawn uppercase
	Meta    string // optional right-aligned detail (counts, state)
	Focused bool
	Width   int
	Height  int // total rows including chrome; 0 = auto height
	Body    string
}

// NewPane builds a pane.
func NewPane(title string, width int, focused bool) Pane {
	return Pane{Title: title, Width: width, Focused: focused}
}

// View renders exactly Width cells per row, and exactly Height rows when
// Height > 0 (shorter bodies are padded, longer ones clipped).
func (p Pane) View() string {
	if p.Width <= 0 {
		return ""
	}

	lines := []string{p.titleRow(), p.ruleRow()}

	body := p.Body
	if body != "" {
		for _, line := range strings.Split(body, "\n") {
			lines = append(lines, padRight(Truncate(line, p.Width), p.Width))
		}
	}

	if p.Height > 0 {
		if len(lines) > p.Height {
			lines = lines[:p.Height]
		}
		for len(lines) < p.Height {
			lines = append(lines, strings.Repeat(" ", p.Width))
		}
	}
	return strings.Join(lines, "\n")
}

// titleRow lays out the bar, the name and the optional right-hand meta.
// Meta is dropped before the name is, and the name is truncated before
// either: a pane without its name is unreadable.
func (p Pane) titleRow() string {
	barStyle, titleStyle := styles.PaneBarStyle, styles.PaneTitleStyle
	if p.Focused {
		barStyle, titleStyle = styles.PaneBarFocusStyle, styles.PaneTitleFocusStyle
	}

	title := strings.ToUpper(p.Title)
	meta := p.Meta

	// bar + space + title … meta
	const lead = 2
	avail := p.Width - lead
	if avail < 1 {
		return titleStyle.Render(Truncate(title, p.Width))
	}

	if meta != "" && ansi.StringWidth(meta)+ansi.StringWidth(title)+2 > avail {
		meta = ""
	}
	metaW := 0
	if meta != "" {
		metaW = ansi.StringWidth(meta) + 2 // two spaces before it
	}

	title = Truncate(title, avail-metaW)
	rendered := barStyle.Render(glyphs.Current().Bar) + " " + titleStyle.Render(title)

	if meta != "" {
		rendered += strings.Repeat(" ", avail-metaW-ansi.StringWidth(title)) + styles.MetaStyle.Render(meta)
	}
	return padRight(rendered, p.Width)
}

// ruleRow draws the hairline under the title: accent when the pane holds
// focus, a quiet surface tone otherwise.
func (p Pane) ruleRow() string {
	g := glyphs.Current()
	st := styles.RuleStyle
	if p.Focused {
		st = styles.RuleFocusStyle
	}
	if g.Rule == "" {
		return strings.Repeat(" ", p.Width)
	}
	return Truncate(st.Render(strings.Repeat(g.Rule, p.Width)), p.Width)
}

// padRight pads (or truncates) a rendered line to exactly width cells.
func padRight(s string, width int) string {
	w := ansi.StringWidth(s)
	if w == width {
		return s
	}
	if w > width {
		return Truncate(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}
