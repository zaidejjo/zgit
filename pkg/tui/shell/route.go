// Package shell owns TUI navigation and layout: the screen router, the
// responsive layout computation, and the key→screen registry. Extracted
// from the root model with identical arithmetic (no visual change).
package shell

// ScreenID identifies a top-level view. Negative Panels keeps the old
// fsNone convention; fullscreen IDs stay compatible with subscriber
// event view numbers.
type ScreenID int

const (
	Panels    ScreenID = -1
	PRs       ScreenID = 3
	Issues    ScreenID = 4
	Dashboard ScreenID = 5
	Actions   ScreenID = 6
	Notes     ScreenID = 7
	Worktrees ScreenID = 8
	Stash     ScreenID = 9
	Refs      ScreenID = 10
)

// Tab binds a screen to its header label and hotkey.
type Tab struct {
	ID    ScreenID
	Label string
	Key   string
}

// Tabs is the single source of truth for the header strip, the number
// keys, and palette navigation dispatch.
var Tabs = []Tab{
	{ID: PRs, Label: "PRs", Key: "3"},
	{ID: Issues, Label: "Issues", Key: "4"},
	{ID: Dashboard, Label: "Repos", Key: "5"},
	{ID: Actions, Label: "Actions", Key: "6"},
	{ID: Notes, Label: "Notes", Key: "7"},
	{ID: Worktrees, Label: "Worktrees", Key: "8"},
	{ID: Stash, Label: "Stash", Key: "9"},
	{ID: Refs, Label: "Refs", Key: "0"},
}

// Short returns the compact tab label for narrow terminals ("5R").
func (t Tab) Short() string {
	for _, r := range t.Label {
		return t.Key + string(r)
	}
	return t.Key
}

// Full returns the standard tab label ("5 Repos").
func (t Tab) Full() string {
	return t.Key + " " + t.Label
}

// ByKey returns the tab for a hotkey, or nil.
func ByKey(key string) *Tab {
	for i := range Tabs {
		if Tabs[i].Key == key {
			return &Tabs[i]
		}
	}
	return nil
}

// Known reports whether id is a fullscreen screen (not Panels).
func Known(id ScreenID) bool {
	for _, t := range Tabs {
		if t.ID == id {
			return true
		}
	}
	return false
}
