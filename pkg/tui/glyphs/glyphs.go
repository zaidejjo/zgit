// Package glyphs owns the TUI's symbol vocabulary.
//
// The same design renders in three symbol sets, so a terminal with a
// patched font gets icons while a plain one still gets a coherent
// layout:
//
//	nerd    Nerd Font symbols and Powerline separators
//	unicode box-drawing, blocks, geometric marks (safe everywhere)
//	ascii   pure ASCII, for a font that renders no symbols at all
//
// Nothing here renders; views read a Set and draw it through the width
// law in components (Truncate/RowLine), so a wider glyph can never
// widen a row.
package glyphs

import (
	"os"
	"strings"
)

// Mode selects a symbol set.
type Mode string

// Modes, most to least capable.
const (
	// Auto lets Detect pick from the environment.
	Auto Mode = "auto"
	// Nerd uses Nerd Font symbols and Powerline separators.
	Nerd Mode = "nerd"
	// Unicode uses box-drawing and geometric marks; safe in every
	// monospace font. Default.
	Unicode Mode = "unicode"
	// ASCII uses only 7-bit characters.
	ASCII Mode = "ascii"
)

// Set is one symbol vocabulary. Fields are only the glyphs the UI
// actually draws; a new symbol is added here rather than hardcoded in a
// view.
type Set struct {
	Mode Mode

	// Structure
	Rule     string // horizontal rule
	VRule    string // vertical separator between inline items
	Bar      string // accent bar on the focused pane's title row
	Dash     string // inline separator between key hints
	Ellipsis string // truncation marker

	// Commit graph
	Commit string // ordinary commit
	Merge  string // merge commit
	Lane   string // lane continuing past this row
	Join   string // lane merging into this row

	// Repository marks
	Branch   string
	Tag      string
	Head     string
	Stash    string
	Worktree string
	Star     string

	// Working-tree and review states
	Staged    string
	Unstaged  string
	Untracked string
	Deleted   string
	Draft     string

	// Signals
	Check string
	Cross string
	Warn  string
	Info  string
	Dot   string

	// Arrows
	Right string
	Up    string
	Down  string

	// Powerline separators (empty unless the font has them).
	PillRight string
	PillLeft  string
}

// Sets is the whole vocabulary, indexed by mode.
var Sets = map[Mode]Set{
	Nerd: {
		Mode: Nerd,

		Rule:     "─",
		VRule:    "│",
		Bar:      "▎",
		Dash:     "·",
		Ellipsis: "…",

		// The graph stays geometric: a filled dot reads better at speed
		// than an icon, and every font that carries the rest of the set
		// carries box drawing too.
		Commit: "●",
		Merge:  "\uf126", // code-fork
		Lane:   "│",
		Join:   "┤",

		Branch:   "\ue725", // git branch
		Tag:      "\uf412", // octicon tag
		Head:     "\uf0da", // arrow into HEAD
		Stash:    "\uf187", // archive
		Worktree: "\uf1bb", // tree
		Star:     "\uf005", // star

		Staged:    "\uf00c", // check
		Unstaged:  "\uf044", // pencil
		Untracked: "\uf128", // question
		Deleted:   "\uf1f8", // trash
		Draft:     "\uf111", // dot

		Check: "\uf00c",
		Cross: "\uf00d",
		Warn:  "\uf071",
		Info:  "\uf05a",
		Dot:   "\uf111",

		Right: "→",
		Up:    "↑",
		Down:  "↓",

		PillRight: "\ue0b0",
		PillLeft:  "\ue0b2",
	},
	Unicode: {
		Mode: Unicode,

		Rule:     "─",
		VRule:    "│",
		Bar:      "▎",
		Dash:     "·",
		Ellipsis: "…",

		Commit: "●",
		Merge:  "◆",
		Lane:   "│",
		Join:   "┤",

		Branch:   "",
		Tag:      "▪",
		Head:     "*",
		Stash:    "≡",
		Worktree: "",
		Star:     "★",

		Staged:    "●",
		Unstaged:  "◐",
		Untracked: "○",
		Deleted:   "✕",
		Draft:     "◌",

		Check: "✓",
		Cross: "✗",
		Warn:  "▲",
		Info:  "ℹ",
		Dot:   "•",

		Right: "→",
		Up:    "↑",
		Down:  "↓",
	},
	ASCII: {
		Mode: ASCII,

		Rule:     "-",
		VRule:    "|",
		Bar:      "|",
		Dash:     "-",
		Ellipsis: "...",

		Commit: "o",
		Merge:  "*",
		Lane:   "|",
		Join:   "+",

		Branch:   "b",
		Tag:      "t",
		Head:     "*",
		Stash:    "s",
		Worktree: "w",
		Star:     "+",

		Staged:    "+",
		Unstaged:  "~",
		Untracked: "?",
		Deleted:   "x",
		Draft:     "d",

		Check: "v",
		Cross: "x",
		Warn:  "!",
		Info:  "i",
		Dot:   "-",

		Right: "->",
		Up:    "^",
		Down:  "v",
	},
}

// mode is the resolved mode; Unicode until Detect or Use runs.
var mode = Unicode

// Current returns the active symbol set.
func Current() Set {
	return Sets[mode]
}

// Use selects a mode explicitly. Auto is resolved against the process
// environment.
func Use(m Mode) {
	mode = Resolve(m)
}

// Resolve maps Auto to a concrete mode using the environment, or returns
// m unchanged for an explicit mode. Unknown modes resolve to Unicode.
func Resolve(m Mode) Mode {
	switch m {
	case Nerd, Unicode, ASCII:
		return m
	case "", Auto:
		return Detect(os.Getenv)
	default:
		return Unicode
	}
}

// Parse maps a user-supplied string ("nerd", "nerdfont", "plain"…) to a
// mode. ok is false when the string selects nothing.
func Parse(s string) (Mode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "nerd", "nerdfont", "nerd-font", "nerd_font":
		return Nerd, true
	case "unicode", "utf8", "utf-8", "safe":
		return Unicode, true
	case "ascii", "plain":
		return ASCII, true
	case "auto", "":
		return Auto, true
	default:
		return "", false
	}
}

// Detect guesses the mode from the environment.
//
// A font cannot be queried, so the guess is conservative: only terminals
// that ship a Nerd Font symbol fallback on by default, or an explicit
// signal, select Nerd. Everything else gets Unicode, which is legible in
// any monospace font — boxes in the UI are a worse failure than missing
// icons. ZGIT_TUI_GLYPHS and the tui.glyphs config key override this.
func Detect(getenv func(string) string) Mode {
	if m, ok := Parse(getenv("ZGIT_TUI_GLYPHS")); ok && m != Auto {
		return m
	}
	// Explicit signals from the user's shell profile.
	if getenv("NERD_FONT") != "" || getenv("NERDFONT") != "" {
		return Nerd
	}
	// Terminals that bundle a Nerd Font symbol fallback, so the glyphs
	// resolve even when the user's own font does not carry them.
	//
	// Every other terminal gets Unicode on purpose: a font cannot be
	// queried, and a box glyph in the UI is a worse failure than a
	// missing icon. kitty, iTerm2, VS Code and friends are deliberately
	// absent — they do not bundle symbols.
	switch strings.ToLower(getenv("TERM_PROGRAM")) {
	case "ghostty", "wezterm":
		return Nerd
	}
	return Unicode
}

// EnvVar is the environment variable that forces a mode.
const EnvVar = "ZGIT_TUI_GLYPHS"
