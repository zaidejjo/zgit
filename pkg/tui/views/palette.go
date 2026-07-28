package views

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// SelectionStyle is the highlighted item style.
var SelectionStyle = lipgloss.NewStyle().Background(styles.Mauve).Foreground(styles.Base).Bold(true)

// accentStyle is for keybinding hints.
var accentStyle = lipgloss.NewStyle().Foreground(styles.Mauve)

// mutedStyle is for secondary text.
var mutedStyle = lipgloss.NewStyle().Foreground(styles.Subtext)

const maxVisibleResults = 7

// PaletteCommand describes one command in the palette.
type PaletteCommand struct {
	ID          string // unique identifier
	Label       string // display name
	Description string // help text
	Keys        string // keyboard shortcut hint
}

// PaletteState tracks the palette dialog.
type PaletteState int

const (
	PaletteClosed PaletteState = iota
	PaletteOpen
)

// PaletteModel is a fuzzy-searchable command palette.
type PaletteModel struct {
	State    PaletteState
	Input    string
	Cursor   int
	Commands []PaletteCommand
	Filtered []int // indices into Commands that match query
	Selected int   // index within Filtered
	Width    int
}

// NewPaletteModel creates a palette with default commands.
func NewPaletteModel() PaletteModel {
	p := PaletteModel{}
	p.SetDefaultCommands()
	return p
}

// SetDefaultCommands populates the command list.
func (m *PaletteModel) SetDefaultCommands() {
	m.Commands = []PaletteCommand{
		// Git actions
		{ID: "stage-all", Label: "Stage All", Description: "Stage all unstaged files", Keys: "a"},
		{ID: "unstage-all", Label: "Unstage All", Description: "Unstage all staged files", Keys: "A"},
		{ID: "commit", Label: "Commit", Description: "Open commit dialog", Keys: "c"},
		{ID: "commit-push", Label: "Commit & Push", Description: "Commit and push to remote", Keys: "P"},
		{ID: "cherry-pick", Label: "Cherry-pick", Description: "Cherry-pick commit in log panel", Keys: "C"},
		{ID: "new-branch", Label: "New Branch", Description: "Create a new branch"},
		{ID: "merge-branch", Label: "Merge Branch", Description: "Merge branch into current"},
		{ID: "delete-branch", Label: "Delete Branch", Description: "Delete a branch"},

		// AI
		{ID: "ai-ask", Label: "AI Ask Sidebar", Description: "Open AI Q&A sidebar", Keys: "C-g"},
		{ID: "ai-agent", Label: "AI Agent Sidebar", Description: "Open AI Agent with git commands", Keys: "C-e"},

		// Config
		{ID: "open-config", Label: "Open Config", Description: "Configure GitHub token / AI settings", Keys: "C-t"},

		// Navigation
		{ID: "next-panel", Label: "Next Panel", Description: "Cycle focus to next panel", Keys: "Tab"},
		{ID: "prev-panel", Label: "Prev Panel", Description: "Cycle focus to previous panel", Keys: "S-Tab"},
		{ID: "help", Label: "Toggle Help", Description: "Show interactive keybindings help", Keys: "?"},
		{ID: "refresh", Label: "Refresh State", Description: "Re-fetch all git state", Keys: "R"},

		// System
		{ID: "quit", Label: "Quit", Description: "Exit zgit", Keys: "q"},
	}
}

// Open shows the palette.
func (m *PaletteModel) Open() {
	m.State = PaletteOpen
	m.Input = ""
	m.Cursor = 0
	m.Filtered = nil
	m.Selected = 0
	m.filter()
}

// Close dismisses the palette.
func (m *PaletteModel) Close() {
	m.State = PaletteClosed
}

// Active returns true if palette is open.
func (m *PaletteModel) Active() bool {
	return m.State == PaletteOpen
}

// GetInput returns trimmed input.
func (m *PaletteModel) GetInput() string {
	return strings.TrimSpace(m.Input)
}

// SelectedCommand returns the currently selected command, if any.
func (m *PaletteModel) SelectedCommand() *PaletteCommand {
	if len(m.Filtered) == 0 {
		return nil
	}
	if m.Selected < 0 || m.Selected >= len(m.Filtered) {
		return nil
	}
	idx := m.Filtered[m.Selected]
	return &m.Commands[idx]
}

// HandleKey processes key presses. Returns handled=true.
func (m *PaletteModel) HandleKey(key string) bool {
	if !m.Active() {
		return false
	}

	switch key {
	case "esc":
		m.Close()
		return true
	case "enter":
		return true // signal app to execute
	case "up", "shift+tab":
		if m.Selected > 0 {
			m.Selected--
		}
	case "down", "tab":
		if m.Selected < len(m.Filtered)-1 {
			m.Selected++
		}
	case "backspace":
		if m.Cursor > 0 {
			_, size := utf8.DecodeLastRuneInString(m.Input[:m.Cursor])
			m.Input = m.Input[:m.Cursor-size] + m.Input[m.Cursor:]
			m.Cursor -= size
			m.filter()
		}
	case "left":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "right":
		if m.Cursor < len(m.Input) {
			m.Cursor++
		}
	case "home":
		m.Cursor = 0
	case "end":
		m.Cursor = len(m.Input)
	case " ":
		m.Input = m.Input[:m.Cursor] + " " + m.Input[m.Cursor:]
		m.Cursor++
		m.filter()
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.Input = m.Input[:m.Cursor] + key + m.Input[m.Cursor:]
			m.Cursor++
			m.filter()
		}
	}
	return true
}

// filter updates the filtered results based on input query.
func (m *PaletteModel) filter() {
	query := strings.ToLower(m.Input)
	m.Filtered = nil

	for i, cmd := range m.Commands {
		if query == "" {
			m.Filtered = append(m.Filtered, i)
			continue
		}
		label := strings.ToLower(cmd.Label)
		id := strings.ToLower(cmd.ID)
		if fuzzyMatch(query, label) || fuzzyMatch(query, id) {
			m.Filtered = append(m.Filtered, i)
		}
	}

	if m.Selected >= len(m.Filtered) {
		m.Selected = len(m.Filtered) - 1
	}
	if m.Selected < 0 {
		m.Selected = 0
	}
}

// fuzzyMatch returns true if all chars in query appear in str in order.
func fuzzyMatch(query, str string) bool {
	qi := 0
	for si := 0; si < len(str) && qi < len(query); si++ {
		if str[si] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

// View renders the command palette overlay with scrollable item list and accent selection.
func (m PaletteModel) View(width int) string {
	if !m.Active() {
		return ""
	}

	dialogW := width - 8
	if dialogW < 40 {
		dialogW = 40
	}
	if dialogW > 80 {
		dialogW = 80
	}

	var b strings.Builder

	// Input line
	display := m.Input
	if m.Cursor < len(display) {
		display = display[:m.Cursor] + "█" + display[m.Cursor:]
	} else {
		display += " "
	}

	// Title + input
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render("  Command Palette"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(styles.Subtext).
		Render("  " + display))
	b.WriteString("\n")

	// Scrollable results window
	// Calculate visible window based on selection position
	scrollOffset := 0
	if m.Selected >= maxVisibleResults {
		scrollOffset = m.Selected - maxVisibleResults + 1
	}
	results := m.Filtered
	endIdx := scrollOffset + maxVisibleResults
	if endIdx > len(results) {
		endIdx = len(results)
		scrollOffset = endIdx - maxVisibleResults
		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}
	if scrollOffset < len(results) {
		results = results[scrollOffset:endIdx]
	}

	// Available width for content (dialog minus 4 for border + padding)
	contentW := dialogW - 4
	if contentW < 30 {
		contentW = 30
	}

	for i, idx := range results {
		cmd := m.Commands[idx]
		globalIdx := scrollOffset + i
		isSelected := globalIdx == m.Selected

		// Keybinding hint, right-aligned
		keysStr := ""
		if cmd.Keys != "" {
			keysStr = "  " + accentStyle.Render(cmd.Keys)
		}

		// Build label with prefix
		prefix := "  "
		if isSelected {
			prefix = "❯ "
		}
		label := prefix + cmd.Label

		// Truncate label + keys to fit content width
		labelWidth := lipgloss.Width(label)
		keysWidth := lipgloss.Width(keysStr)
		totalWidth := labelWidth + keysWidth
		if totalWidth > contentW {
			// Truncate label
			overflow := totalWidth - contentW
			labelRunes := []rune(cmd.Label)
			maxLabel := len(labelRunes) - overflow
			if maxLabel < 2 {
				maxLabel = 2
			}
			label = prefix + string(labelRunes[:maxLabel-1]) + "…"
		}

		// Line with 2-column: label left, keys right
		line := label
		if keysStr != "" {
			// Right-align keys
			padding := contentW - lipgloss.Width(label) - lipgloss.Width(keysStr)
			if padding > 0 {
				line += strings.Repeat(" ", padding) + keysStr
			} else {
				line += " " + keysStr
			}
		}

		if isSelected {
			b.WriteString(SelectionStyle.Render(line))
			b.WriteString("\n")
			// Description on next line, indented
			desc := mutedStyle.Render("  " + cmd.Description)
			b.WriteString(desc)
			b.WriteString("\n")
		} else {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if len(results) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render("  No matching commands"))
		b.WriteString("\n")
	}

	// Hint
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Overlay0).Render("  Type to filter  ↑↓=navigate  Enter=execute  Esc=close"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(dialogW).
		Padding(0, 1).
		Background(styles.Surface).
		Render(b.String())
}
