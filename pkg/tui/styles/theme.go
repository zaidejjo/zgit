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

	// Cockpit chrome
	PaneBarStyle        lipgloss.Style
	PaneBarFocusStyle   lipgloss.Style
	PaneTitleStyle      lipgloss.Style
	PaneTitleFocusStyle lipgloss.Style
	RuleStyle           lipgloss.Style
	RuleFocusStyle      lipgloss.Style
	MetaStyle           lipgloss.Style
	FaintStyle          lipgloss.Style

	// Header strip
	HeaderTabStyle       lipgloss.Style
	HeaderTabActiveStyle lipgloss.Style
	HeaderTabBadgeStyle  lipgloss.Style
	HeaderRuleStyle      lipgloss.Style

	// Footer
	HintKeyStyle   lipgloss.Style
	HintDescStyle  lipgloss.Style
	HintSepStyle   lipgloss.Style
	StatusBarText  lipgloss.Style
	StatusBarMuted lipgloss.Style
	SegmentAccent  lipgloss.Style
	SegmentNeutral lipgloss.Style
)

// Refresh rebuilds every style from the current tokens.
// Call after SetLight/SetDark.
func Refresh() {
	// App-level. No fixed width: the frame is sized per render from the
	// terminal, a hardcoded 80 only ever fit an 80-column terminal.
	AppStyle = lipgloss.NewStyle().
		Padding(0, 1)

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
	// Tab strip: only the active screen carries a fill, so the eye lands
	// on it instead of on a row of equal-weight blocks. The hotkey is a
	// dim prefix, not part of the label.
	HeaderTabStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(Subtext)
	HeaderTabActiveStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(Accent).
		Foreground(Base).
		Bold(true)
	HeaderTabBadgeStyle = lipgloss.NewStyle().
		Foreground(Peach)
	HeaderRuleStyle = lipgloss.NewStyle().
		Foreground(Rule)

	// Cockpit pane chrome
	PaneBarStyle = lipgloss.NewStyle().Foreground(Rule)
	PaneBarFocusStyle = lipgloss.NewStyle().Foreground(Accent)
	PaneTitleStyle = lipgloss.NewStyle().Foreground(PaneTitle).Bold(true)
	PaneTitleFocusStyle = lipgloss.NewStyle().Foreground(PaneTitleFocus).Bold(true)
	RuleStyle = lipgloss.NewStyle().Foreground(Rule)
	RuleFocusStyle = lipgloss.NewStyle().Foreground(Accent)
	MetaStyle = lipgloss.NewStyle().Foreground(Overlay0)
	FaintStyle = lipgloss.NewStyle().Foreground(Rule)

	// Content areas
	ContentStyle = lipgloss.NewStyle().
		Padding(0, 1).
		MarginTop(0)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
		Background(Surface).
		Foreground(Subtext).
		Padding(0, 1).
		Height(1)
	StatusBarBranchStyle = lipgloss.NewStyle().
		Foreground(Teal).
		Bold(true)
	StatusBarInfoStyle = lipgloss.NewStyle().
		Foreground(Subtext)
	StatusBarText = lipgloss.NewStyle().
		Foreground(Text).
		Bold(true)
	StatusBarMuted = lipgloss.NewStyle().
		Foreground(Overlay0)
	HintStyle = lipgloss.NewStyle().
		Foreground(Overlay0)
	HintKeyStyle = lipgloss.NewStyle().
		Foreground(Subtext).
		Bold(true)
	HintDescStyle = lipgloss.NewStyle().
		Foreground(Overlay0)
	HintSepStyle = lipgloss.NewStyle().
		Foreground(Rule)
	SegmentAccent = lipgloss.NewStyle().
		Background(Accent).
		Foreground(Base).
		Bold(true)
	SegmentNeutral = lipgloss.NewStyle().
		Background(ChipBg).
		Foreground(Text)

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
		Background(DiffAddBg)
	DiffDelStyle = lipgloss.NewStyle().
		Foreground(Red).
		Background(DiffDelBg)
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
