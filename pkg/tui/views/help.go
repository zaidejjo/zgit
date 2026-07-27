package views

import (
	"strings"

	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// HelpModel renders the keybindings help screen.
type HelpModel struct{}

// View renders the full help text.
func (h HelpModel) View(width int) string {
	var b strings.Builder

	b.WriteString(styles.DialogTitleStyle.Render("  Keybindings"))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		keys  []helpEntry
	}{
		{
			title: "Navigation",
			keys: []helpEntry{
				{"↑ / k", "Move up"},
				{"↓ / j", "Move down"},
				{"g / home", "Go to top"},
				{"G / end", "Go to bottom"},
				{"PgUp / C-b", "Page up"},
				{"PgDn / C-f", "Page down"},
			},
		},
		{
			title: "Panel Focus",
			keys: []helpEntry{
				{"Tab", "Next panel (Status→Log→Branches)"},
				{"S-Tab", "Previous panel"},
			},
		},
		{
			title: "Full-Screen Views",
			keys: []helpEntry{
				{"3", "Pull Requests"},
				{"4", "Issues"},
				{"Tab / Esc", "Back to panels"},
			},
		},
		{
			title: "File Operations (Status panel)",
			keys: []helpEntry{
				{"space", "Stage / Unstage toggle"},
				{"s", "Stage file"},
				{"S", "Unstage file"},
				{"a", "Stage all"},
				{"A", "Unstage all"},
				{"d", "Discard file changes"},
				{"enter", "View diff (on file)"},
				{"c", "Open commit dialog"},
				{"P", "Commit & Push"},
			},
		},
		{
			title: "Log Operations (Log panel)",
			keys: []helpEntry{
				{"enter", "View commit diff"},
				{"C", "Cherry-pick commit (stages, no commit)"},
				{"r", "Rebase (coming soon)"},
			},
		},
		{
			title: "Branch Operations (Branches panel)",
			keys: []helpEntry{
				{"c / enter", "Checkout branch"},
				{"n", "New branch"},
				{"x", "Delete branch"},
				{"m", "Merge branch"},
			},
		},
		{
			title: "Commit Dialog",
			keys: []helpEntry{
				{"enter", "Next field / Confirm"},
				{"Ctrl+D", "Finish description"},
				{"Esc", "Cancel / Close"},
			},
		},
		{
			title: "Global",
			keys: []helpEntry{
				{"q / C-c", "Quit"},
				{"r", "Refresh"},
				{"?", "Toggle this help"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString(styles.SectionTitleStyle.Render("  " + section.title))
		b.WriteString("\n")

		maxKeyLen := 0
		for _, e := range section.keys {
			if len(e.key) > maxKeyLen {
				maxKeyLen = len(e.key)
			}
		}
		if maxKeyLen < 12 {
			maxKeyLen = 12
		}

		for _, e := range section.keys {
			padding := maxKeyLen - len(e.key)
			if padding < 1 {
				padding = 1
			}
			keyPart := styles.HelpKeyStyle.Render(" " + e.key + strings.Repeat(" ", padding))
			descPart := styles.HelpDescStyle.Render(e.desc)
			b.WriteString("  " + keyPart + descPart + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(styles.SubtitleStyle.Render("  Press ? again to close help"))
	b.WriteString("\n")

	return b.String()
}

type helpEntry struct {
	key  string
	desc string
}
