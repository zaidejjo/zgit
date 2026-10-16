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

	// Semantic aliases (re-pointed on theme switch)
	Success   = Green
	Warning   = Yellow
	Danger    = Red
	Info      = Blue
	Accent    = Mauve
	Selection = lipgloss.Color("#45475a")

	// Panel border colors
	ActivePanelBorderColor   = Accent
	InactivePanelBorderColor = Overlay0

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
	Success = Green
	Warning = Yellow
	Danger = Red
	Info = Blue
	Accent = Mauve
	Selection = lipgloss.Color("#acb0be")
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
	Success = Green
	Warning = Yellow
	Danger = Red
	Info = Blue
	Accent = Mauve
	Selection = lipgloss.Color("#45475a")
	ActivePanelBorderColor = Accent
	InactivePanelBorderColor = Overlay0
	Dark = true
}
