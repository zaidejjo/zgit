package components

import (
	"strings"
	"time"

	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Tone selects the toast accent.
type Tone int

const (
	ToneInfo Tone = iota
	ToneSuccess
	ToneError
)

// Toast is one transient notification.
type Toast struct {
	Text    string
	Tone    Tone
	Expires time.Time
}

// Toasts is a small expiring stack (cap 3, oldest dropped first).
type Toasts struct {
	items []Toast
	ttl   time.Duration
}

// NewToasts creates a stack with the given per-toast lifetime.
func NewToasts(ttl time.Duration) Toasts {
	return Toasts{ttl: ttl}
}

// Push adds a toast, dropping the oldest past the cap.
func (t *Toasts) Push(text string, tone Tone) {
	t.items = append(t.items, Toast{Text: text, Tone: tone, Expires: time.Now().Add(t.ttl)})
	for len(t.items) > 3 {
		t.items = t.items[1:]
	}
}

// Expire drops elapsed toasts. True when anything remains.
func (t *Toasts) Expire() bool {
	now := time.Now()
	kept := t.items[:0]
	for _, it := range t.items {
		if it.Expires.After(now) {
			kept = append(kept, it)
		}
	}
	t.items = kept
	return len(t.items) > 0
}

// Empty reports no visible toasts.
func (t *Toasts) Empty() bool {
	return len(t.items) == 0
}

// View renders the stack as plain lines. The shell prints them above
// the status bar (brief reflow while visible — safer than splicing
// into ANSI-styled frame lines).
func (t *Toasts) View() string {
	var b strings.Builder
	for _, it := range t.items {
		mark := "◌"
		style := styles.SubtitleStyle
		switch it.Tone {
		case ToneSuccess:
			mark = "✓"
			style = styles.StatusStagedStyle
		case ToneError:
			mark = "✗"
			style = styles.ErrorStyle
		}
		b.WriteString(style.Render(mark+" "+it.Text) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
