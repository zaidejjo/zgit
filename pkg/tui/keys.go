package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the TUI app.
// Vim-friendly navigation with consistent cross-view bindings.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	First    key.Binding
	Last     key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// View switching
	TabNext   key.Binding
	TabPrev   key.Binding
	TabStatus key.Binding
	TabLog    key.Binding
	TabBranch key.Binding

	// Actions
	Enter  key.Binding
	Escape key.Binding
	Space  key.Binding

	// File operations
	Stage      key.Binding
	Unstage    key.Binding
	StageAll   key.Binding
	UnstageAll key.Binding
	Discard    key.Binding

	// Branch operations
	Checkout    key.Binding
	BranchNew   key.Binding
	BranchDel   key.Binding
	BranchMerge key.Binding

	// Commit
	Commit key.Binding
	Amend  key.Binding

	// Global
	Help    key.Binding
	Refresh key.Binding
	Quit    key.Binding
}

// DefaultKeyMap returns the standard keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		First:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g/home", "top")),
		Last:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G/end", "bottom")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+b"), key.WithHelp("PgUp", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+f"), key.WithHelp("PgDn", "page down")),

		// View switching
		TabNext:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next view")),
		TabPrev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev view")),
		TabStatus: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "status")),
		TabLog:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "log")),
		TabBranch: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "branches")),

		// Actions
		Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Escape: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Space:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),

		// File operations
		Stage:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stage")),
		Unstage:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "unstage")),
		StageAll:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "stage all")),
		UnstageAll: key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "unstage all")),
		Discard:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "discard")),

		// Branch operations
		Checkout:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "checkout")),
		BranchNew:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new branch")),
		BranchDel:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete branch")),
		BranchMerge: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merge")),

		// Commit
		Commit: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commit")),
		Amend:  key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "amend")),

		// Global
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// FullHelp returns all keybindings grouped by category.
func (k KeyMap) FullHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.First, k.Last, k.PageUp, k.PageDown,
		k.TabNext, k.TabPrev,
		k.Enter, k.Escape, k.Space,
		k.Stage, k.Unstage, k.StageAll, k.UnstageAll, k.Discard,
		k.Commit, k.Amend,
		k.Checkout, k.BranchNew, k.BranchDel, k.BranchMerge,
		k.Refresh, k.Help, k.Quit,
	}
}
