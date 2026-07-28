package views

import (
	"strings"
	"unicode/utf8"

	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// HelpModel renders the interactive keybindings help with fuzzy search.
type HelpModel struct {
	Input     string
	Cursor    int
	ShowInput bool // true when user has typed something
	Width     int
}

// helpEntry is a single keybinding row.
type helpEntry struct {
	key  string
	desc string
}

// helpSection groups related keybindings.
type helpSection struct {
	title string
	keys  []helpEntry
}

// allSections returns all help sections.
func allSections() []helpSection {
	return []helpSection{
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
			title: "AI",
			keys: []helpEntry{
				{"C-g", "Ask AI a question about the repo"},
				{"C-e", "AI Agent panel (git commands)"},
				{"/cmd", "Slash commands: /models, /chats, /new"},
			},
		},
		{
			title: "Config",
			keys: []helpEntry{
				{"C-t", "Open config (GitHub token, AI settings)"},
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
				{"R", "Refresh stale data"},
				{"C-p", "Command palette"},
				{"?", "Toggle this help"},
			},
		},
	}
}

// searchableEntry joins key+desc for text matching.
func (e helpEntry) searchText() string {
	return strings.ToLower(e.key + " " + e.desc)
}

// fuzzyMatchHelp returns true if query chars appear in str in order.
func fuzzyMatchHelp(query, str string) bool {
	qi := 0
	for si := 0; si < len(str) && qi < len(query); si++ {
		if str[si] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

// HandleKey processes a key press. Returns handled=true.
func (h *HelpModel) HandleKey(key string) bool {
	switch key {
	case "esc", "?":
		return false // signal caller to close
	case "backspace":
		if h.Cursor > 0 {
			_, size := utf8.DecodeLastRuneInString(h.Input[:h.Cursor])
			h.Input = h.Input[:h.Cursor-size] + h.Input[h.Cursor:]
			h.Cursor -= size
			h.ShowInput = len(h.Input) > 0
		}
	case "left":
		if h.Cursor > 0 {
			h.Cursor--
		}
	case "right":
		if h.Cursor < len(h.Input) {
			h.Cursor++
		}
	case "home":
		h.Cursor = 0
	case "end":
		h.Cursor = len(h.Input)
	case " ":
		h.Input = h.Input[:h.Cursor] + " " + h.Input[h.Cursor:]
		h.Cursor++
		h.ShowInput = true
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			h.Input = h.Input[:h.Cursor] + key + h.Input[h.Cursor:]
			h.Cursor++
			h.ShowInput = true
		}
	}
	return true
}

// View renders the help screen with optional search filter.
func (h HelpModel) View(width int) string {
	var b strings.Builder

	// Search bar at top
	searchBar := "Search: "
	if h.ShowInput {
		display := h.Input
		if h.Cursor < len(display) {
			display = display[:h.Cursor] + "█" + display[h.Cursor:]
		}
		searchBar += display
	} else {
		searchBar += "type to filter keys..."
	}
	b.WriteString(styles.DialogTitleStyle.Render("  Keybindings"))
	b.WriteString("\n")
	b.WriteString(styles.SubtitleStyle.Render("  " + searchBar))
	b.WriteString("\n\n")

	query := strings.ToLower(h.Input)
	resultCount := 0

	for _, section := range allSections() {
		// Filter entries
		var matching []helpEntry
		for _, e := range section.keys {
			if query == "" || fuzzyMatchHelp(query, e.searchText()) {
				matching = append(matching, e)
			}
		}
		if len(matching) == 0 {
			continue
		}

		// Section title
		b.WriteString(styles.SectionTitleStyle.Render("  " + section.title))
		b.WriteString("\n")

		maxKeyLen := 0
		for _, e := range matching {
			if len(e.key) > maxKeyLen {
				maxKeyLen = len(e.key)
			}
		}
		if maxKeyLen < 12 {
			maxKeyLen = 12
		}

		for _, e := range matching {
			padding := maxKeyLen - len(e.key)
			if padding < 1 {
				padding = 1
			}
			keyPart := styles.HelpKeyStyle.Render(" " + e.key + strings.Repeat(" ", padding))
			descPart := styles.HelpDescStyle.Render(e.desc)
			b.WriteString("  " + keyPart + descPart + "\n")
			resultCount++
		}
		b.WriteString("\n")
	}

	if resultCount == 0 && h.ShowInput {
		b.WriteString(styles.SubtitleStyle.Render("  No matching keybindings for \"" + h.Input + "\""))
		b.WriteString("\n\n")
	}

	b.WriteString(styles.SubtitleStyle.Render("  Press ? or Esc to close  |  Type to filter shortcuts"))
	b.WriteString("\n")

	return b.String()
}
