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

	// Panel focus
	PanelNext key.Binding
	PanelPrev key.Binding

	// Full-screen views
	ViewPRs    key.Binding
	ViewIssues key.Binding

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
	Commit     key.Binding
	CommitPush key.Binding
	Amend      key.Binding

	// Log operations
	CherryPick  key.Binding
	RebaseStart key.Binding

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

		// Panel focus
		PanelNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		PanelPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev panel")),

		// Full-screen views
		ViewPRs:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "PRs")),
		ViewIssues: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "issues")),

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
		Commit:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commit")),
		CommitPush: key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "commit & push")),
		Amend:      key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "amend")),

		// Log operations
		CherryPick:  key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "cherry-pick (no commit)")),
		RebaseStart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rebase")),

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
		k.PanelNext, k.PanelPrev,
		k.ViewPRs, k.ViewIssues,
		k.Enter, k.Escape, k.Space,
		k.Stage, k.Unstage, k.StageAll, k.UnstageAll, k.Discard,
		k.Commit, k.CommitPush, k.Amend,
		k.CherryPick, k.RebaseStart,
		k.Checkout, k.BranchNew, k.BranchDel, k.BranchMerge,
		k.Refresh, k.Help, k.Quit,
	}
}
