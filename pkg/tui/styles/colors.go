// Package styles provides the TUI color theme: Catppuccin Mocha (dark)
// with a Latte (light) variant. Colors are variables so the theme can
// switch at runtime; every view reads these tokens, never literals.
package styles

import "github.com/charmbracelet/lipgloss"

// Color palette — dark (Mocha) defaults; SetLight/SetDark swap them.
var (
	Base      = lipgloss.Color("#1e1e2e")
	Surface   = lipgloss.Color("#181825")
	Overlay   = lipgloss.Color("#313244")
	Text      = lipgloss.Color("#cdd6f4")
	Subtext   = lipgloss.Color("#a6adc8")
	Overlay0  = lipgloss.Color("#6c7086")
	Blue      = lipgloss.Color("#89b4fa")
	Green     = lipgloss.Color("#a6e3a1")
	Yellow    = lipgloss.Color("#f9e2af")
	Red       = lipgloss.Color("#f38ba8")
	Mauve     = lipgloss.Color("#cba6f7")
	Peach     = lipgloss.Color("#fab387")
	Teal      = lipgloss.Color("#94e2d5")
	Rosewater = lipgloss.Color("#f5e0dc")

	// Chrome layers. The cockpit look is built on flat panes with rule
	// separators instead of boxes, so it needs the full Catppuccin
	// surface ramp: Crust (app edge), Mantle (bar), Surface0-2 (chips,
	// rules, hairlines).
	Crust    = lipgloss.Color("#11111b")
	Mantle   = lipgloss.Color("#181825")
	Surface0 = lipgloss.Color("#313244")
	Surface1 = lipgloss.Color("#45475a")
	Surface2 = lipgloss.Color("#585b70")

	// Extra hues, so a badge is identifiable by hue and not only by
	// weight (tags, remotes, lanes, review states).
	Sky      = lipgloss.Color("#89dceb")
	Sapphire = lipgloss.Color("#74c7ec")
	Lavender = lipgloss.Color("#b4befe")
	Pink     = lipgloss.Color("#f5c2e7")
	Maroon   = lipgloss.Color("#eba0ac")
	Flamingo = lipgloss.Color("#f2cdcd")

	// Semantic aliases (re-pointed on theme switch)
	Success   = Green
	Warning   = Yellow
	Danger    = Red
	Info      = Blue
	Accent    = Mauve
	Selection = lipgloss.Color("#45475a")

	// Cockpit chrome. Rules are hairlines, never borders: quiet when a
	// pane is inactive, accent when it holds focus.
	Rule           = Surface2
	PaneTitle      = Overlay0
	PaneTitleFocus = Accent
	ChipBg         = Surface0
	ChipBgAccent   = Mauve
	ChipFgAccent   = Base
	TrackAdd       = Green
	TrackDel       = Red

	// Panel border colors
	ActivePanelBorderColor   = Accent
	InactivePanelBorderColor = Overlay0

	// Diff line backgrounds. Tokens, not literals: a dark green would be
	// unreadable the moment the light palette is active.
	DiffAddBg = lipgloss.Color("#1a3a1a")
	DiffDelBg = lipgloss.Color("#3a1a1a")

	// Dark reports whether the dark palette is active.
	Dark = true
)

// SetLight swaps to the Catppuccin Latte palette.
func SetLight() {
	Base = "#eff1f5"
	Surface = "#e6e9ef"
	Overlay = "#ccd0da"
	Text = "#4c4f69"
	Subtext = "#6c6f85"
	Overlay0 = "#9ca0b0"
	Blue = "#1e66f5"
	Green = "#40a02b"
	Yellow = "#df8e1d"
	Red = "#d20f39"
	Mauve = "#8839ef"
	Peach = "#fe640b"
	Teal = "#179299"
	Rosewater = "#dc8a78"
	Crust = "#dce0e8"
	Mantle = "#e6e9ef"
	Surface0 = "#ccd0da"
	Surface1 = "#bcc0cc"
	Surface2 = "#acb0be"
	Sky = "#04a5e5"
	Sapphire = "#209fb5"
	Lavender = "#7287fd"
	Pink = "#ea76cb"
	Maroon = "#e64553"
	Flamingo = "#dd7878"
	Success = Green
	Warning = Yellow
	Danger = Red
	Info = Blue
	Accent = Mauve
	Selection = lipgloss.Color("#acb0be")
	Rule = Surface2
	PaneTitle = Overlay0
	PaneTitleFocus = Accent
	ChipBg = "#ccd0da"
	ChipBgAccent = Mauve
	ChipFgAccent = Base
	TrackAdd = Green
	TrackDel = Red
	DiffAddBg = "#d7f0d7"
	DiffDelBg = "#f7d7dc"
	ActivePanelBorderColor = Accent
	InactivePanelBorderColor = Overlay0
	Dark = false
}

// SetDark restores the Catppuccin Mocha palette.
func SetDark() {
	Base = "#1e1e2e"
	Surface = "#181825"
	Overlay = "#313244"
	Text = "#cdd6f4"
	Subtext = "#a6adc8"
	Overlay0 = "#6c7086"
	Blue = "#89b4fa"
	Green = "#a6e3a1"
	Yellow = "#f9e2af"
	Red = "#f38ba8"
	Mauve = "#cba6f7"
	Peach = "#fab387"
	Teal = "#94e2d5"
	Rosewater = "#f5e0dc"
	Crust = "#11111b"
	Mantle = "#181825"
	Surface0 = "#313244"
	Surface1 = "#45475a"
	Surface2 = "#585b70"
	Sky = "#89dceb"
	Sapphire = "#74c7ec"
	Lavender = "#b4befe"
	Pink = "#f5c2e7"
	Maroon = "#eba0ac"
	Flamingo = "#f2cdcd"
	Success = Green
	Warning = Yellow
	Danger = Red
	Info = Blue
	Accent = Mauve
	Selection = lipgloss.Color("#45475a")
	Rule = Surface2
	PaneTitle = Overlay0
	PaneTitleFocus = Accent
	ChipBg = Surface0
	ChipBgAccent = Mauve
	ChipFgAccent = Base
	TrackAdd = Green
	TrackDel = Red
	DiffAddBg = "#1a3a1a"
	DiffDelBg = "#3a1a1a"
	ActivePanelBorderColor = Accent
	InactivePanelBorderColor = Overlay0
	Dark = true
}
