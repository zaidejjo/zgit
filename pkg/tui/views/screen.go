package views

import (
	"fmt"
	"strings"

	"github.com/zaidejjo/zgit/pkg/tui/components"
)

// screenChrome frames a full-screen list the way the dashboard frames its
// panes: the same title row with an accent bar, the same hairline rule.
// Switching screens must not change the visual grammar, and a full-screen
// list has no neighbour to borrow a border from.
func screenChrome(title, meta, body string, width int) string {
	p := components.NewPane(title, width, true)
	p.Meta = meta
	p.Body = body
	return p.View()
}

// screenMeta renders a screen's right-hand detail: the item count, or the
// visible/total count followed by the filter that is narrowing it. A cap
// keeps a long filter from pushing the pane name out of the title row.
func screenMeta(shown, total int, filter string) string {
	meta := fmt.Sprintf("%d", total)
	if f := strings.TrimSpace(filter); f != "" {
		meta = fmt.Sprintf("%d/%d · %s", shown, total, f)
	}
	return components.Truncate(meta, 44)
}
