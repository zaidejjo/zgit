package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Common style definitions used across all views. Declared once, built
// by Refresh() from the current tokens so SetLight/SetDark take effect.
var (
	AppStyle              lipgloss.Style
	ActiveTabStyle        lipgloss.Style
	InactiveTabStyle      lipgloss.Style
	HeaderLogoStyle       lipgloss.Style
	HeaderMetaStyle       lipgloss.Style
	HeaderBranchStyle     lipgloss.Style
	ContentStyle          lipgloss.Style
	StatusBarStyle        lipgloss.Style
	StatusBarBranchStyle  lipgloss.Style
	StatusBarInfoStyle    lipgloss.Style
	HintStyle             lipgloss.Style
	ListItemStyle         lipgloss.Style
	ListItemSelectedStyle lipgloss.Style
	FilterBarStyle        lipgloss.Style
	ListItemActiveStyle   lipgloss.Style
	SectionTitleStyle     lipgloss.Style
	SubtitleStyle         lipgloss.Style
	StatusStagedStyle     lipgloss.Style
	StatusUnstagedStyle   lipgloss.Style
	StatusUntrackedStyle  lipgloss.Style
	StatusDeletedStyle    lipgloss.Style
	DiffAddStyle          lipgloss.Style
	DiffDelStyle          lipgloss.Style
	DiffHeaderStyle       lipgloss.Style
	HelpKeyStyle          lipgloss.Style
	HelpDescStyle         lipgloss.Style
	LoadingStyle          lipgloss.Style
	ErrorStyle            lipgloss.Style
	DialogBoxStyle        lipgloss.Style
	DialogTitleStyle      lipgloss.Style
)

// Refresh rebuilds every style from the current tokens.
// Call after SetLight/SetDark.
func Refresh() {
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

	// Header bar (top tab strip)
	HeaderLogoStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(Mauve).
		Foreground(Base).
		Bold(true)
	HeaderMetaStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(Subtext)
	HeaderBranchStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(Teal).
		Bold(true)

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
	HintStyle = lipgloss.NewStyle().
		Foreground(Overlay0)

	// Lists
	ListItemStyle = lipgloss.NewStyle().
		Foreground(Text)
	ListItemSelectedStyle = lipgloss.NewStyle().
		PaddingLeft(1).Background(Selection).
		Foreground(Text).
		Bold(true)
	FilterBarStyle = lipgloss.NewStyle().
		Foreground(Yellow)

	ListItemActiveStyle = lipgloss.NewStyle().
		Foreground(Blue).
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
}

func init() {
	Refresh()
}

// Divider returns a horizontal line.
func Divider() string {
	return lipgloss.NewStyle().
		Foreground(Overlay0).
		Render(strings.Repeat("─", 78))
}
