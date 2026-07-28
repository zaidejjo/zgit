// Package styles provides a clean, accessible color theme for the TUI.
// Based on Catppuccin Mocha with good contrast ratios.
package styles

import "github.com/charmbracelet/lipgloss"

// Color palette — accessible, high-contrast, modern.
const (
	Base      = lipgloss.Color("#1e1e2e") // dark background
	Surface   = lipgloss.Color("#181825") // darker surface
	Overlay   = lipgloss.Color("#313244") // elevated surface
	Text      = lipgloss.Color("#cdd6f4") // primary text
	Subtext   = lipgloss.Color("#a6adc8") // secondary text
	Overlay0  = lipgloss.Color("#6c7086") // muted
	Blue      = lipgloss.Color("#89b4fa") // primary accent
	Green     = lipgloss.Color("#a6e3a1") // success
	Yellow    = lipgloss.Color("#f9e2af") // warning
	Red       = lipgloss.Color("#f38ba8") // error/danger
	Mauve     = lipgloss.Color("#cba6f7") // purple accent
	Peach     = lipgloss.Color("#fab387") // orange
	Teal      = lipgloss.Color("#94e2d5") // cyan
	Rosewater = lipgloss.Color("#f5e0dc") // pinkish

	// Semantic aliases
	Success   = Green
	Warning   = Yellow
	Danger    = Red
	Info      = Blue
	Accent    = Mauve
	Selection = lipgloss.Color("#45475a")

	// Panel border colors
	ActivePanelBorderColor   = Accent
	InactivePanelBorderColor = Overlay0
)
