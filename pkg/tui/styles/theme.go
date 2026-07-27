package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Common style definitions used across all views.
var (
	// App-level
	AppStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(80)

	// Tabs
	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(Blue).
			Foreground(Base).
			Bold(true)
	InactiveTabStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Background(Overlay).
				Foreground(Subtext)

	// Content areas
	ContentStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginTop(0)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(Surface).
			Foreground(Subtext).
			Padding(0, 1).
			Width(80).
			Height(1)
	StatusBarBranchStyle = lipgloss.NewStyle().
				Foreground(Teal).
				Bold(true)
	StatusBarInfoStyle = lipgloss.NewStyle().
				Foreground(Subtext)

	// Lists
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(Text)
	ListItemSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Background(Selection).
				Foreground(Text).
				Bold(true)
	ListItemActiveStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Background(Blue).
				Foreground(Base).
				Bold(true)

	// Headers
	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(Blue).
				Bold(true).
				MarginTop(1).
				MarginBottom(1).
				PaddingLeft(1)
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Subtext).
			Italic(true)

	// File status indicators
	StatusStagedStyle = lipgloss.NewStyle().
				Foreground(Green).
				Bold(true)
	StatusUnstagedStyle = lipgloss.NewStyle().
				Foreground(Yellow)
	StatusUntrackedStyle = lipgloss.NewStyle().
				Foreground(Subtext)
	StatusDeletedStyle = lipgloss.NewStyle().
				Foreground(Red).
				Strikethrough(true)

	// Diff
	DiffAddStyle = lipgloss.NewStyle().
			Foreground(Green).
			Background(lipgloss.Color("#1a3a1a"))
	DiffDelStyle = lipgloss.NewStyle().
			Foreground(Red).
			Background(lipgloss.Color("#3a1a1a"))
	DiffHeaderStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	// Help
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)
	HelpDescStyle = lipgloss.NewStyle().
			Foreground(Subtext)

	// Loading
	LoadingStyle = lipgloss.NewStyle().
			Foreground(Subtext).
			Italic(true)

	// Error
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	// Dialog
	DialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(1, 2).
			Background(Surface)
	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(Blue).
				Bold(true)
)

// Divider returns a horizontal line.
func Divider() string {
	return lipgloss.NewStyle().
		Foreground(Overlay0).
		Render(strings.Repeat("─", 78))
}
