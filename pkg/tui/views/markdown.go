package views

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

// mdMemo caches the last rendered result. TUI View() runs on every frame,
// and re-parsing markdown each time would waste cycles on long AI answers.
// Conversations are append-only and width is stable, so a 1-entry cache hits
// almost always.
var mdMemo struct {
	sync.Mutex
	content string
	width   int
	output  string
}

// RenderMarkdown renders GitHub-flavored markdown to ANSI for the terminal.
// Auto style follows the terminal background; output is word-wrapped.
// On any failure it returns the input unchanged — never break the UI
// because rendering failed.
func RenderMarkdown(content string, width int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	if width < 20 {
		width = 80
	}
	if width > 120 {
		width = 120
	}

	mdMemo.Lock()
	defer mdMemo.Unlock()
	if mdMemo.content == content && mdMemo.width == width {
		return mdMemo.output
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	out = strings.TrimRight(out, "\n")

	mdMemo.content = content
	mdMemo.width = width
	mdMemo.output = out
	return out
}
