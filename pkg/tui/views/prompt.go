package views

import (
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Prompt is a one-line inline input row used for quick add-flows
// (new worktree path, stash message, tag name). Lighter than a huh
// dialog: it renders inside the list view and submits on enter.
type Prompt struct {
	Active bool
	Label  string
	Value  string
}

// Open shows the prompt with a label.
func (p *Prompt) Open(label string) {
	p.Active = true
	p.Label = label
	p.Value = ""
}

// TypeKey routes typing. Returns done (enter with any value),
// cancelled (esc), consumed.
// Empty submit is allowed — callers decide requiredness.
func (p *Prompt) TypeKey(key string) (done, cancelled bool) {
	if !p.Active {
		return false, false
	}
	switch key {
	case "enter":
		p.Active = false
		return true, false
	case "esc":
		p.Active = false
		p.Value = ""
		return false, true
	case "backspace":
		if len(p.Value) > 0 {
			p.Value = p.Value[:len(p.Value)-1]
		}
		return false, false
	default:
		if len(key) == 1 || key == " " {
			p.Value += key
		}
		return false, false
	}
}

// View renders the prompt row, or "" when closed.
func (p *Prompt) View() string {
	if !p.Active {
		return ""
	}
	return styles.FilterBarStyle.Render(" "+p.Label+" "+p.Value+"█") + "\n"
}
