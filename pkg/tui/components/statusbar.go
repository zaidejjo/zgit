package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// StatusBarData holds dynamic data for the status bar.
type StatusBarData struct {
	Branch         string
	Ahead          int
	Behind         int
	StagedCount    int
	UnstagedCount  int
	UntrackedCount int
	GhOwner        string
	GhRepo         string
	GhDetected     bool
	GhUser         string // authenticated username, e.g. "octocat"
	AIProvider     string // e.g. "openai:gpt-4o" or empty
	RepoPath       string // absolute path to repository
}

// StatusBar renders the bottom bar: repository state on the left, key
// hints on the right.
//
// The bar is a priority list, not a sentence. Segments are added while
// they fit and dropped from the least useful end, so a narrow terminal
// loses the AI provider before it loses the branch, and the hints — the
// one thing a new user needs — survive longest.
type StatusBar struct {
	Width int
	Data  StatusBarData
	Hints []Hint
}

// NewStatusBar creates a status bar component.
func NewStatusBar(width int, data StatusBarData) StatusBar {
	return StatusBar{
		Width: width,
		Data:  data,
		Hints: DefaultHints(),
	}
}

// DefaultHints is the footer legend, ordered least to most important so
// FitHints drops from the left.
func DefaultHints() []Hint {
	return []Hint{
		{Key: "^p", Desc: "palette"},
		{Key: "L", Desc: "theme"},
		{Key: "tab", Desc: "panes"},
		{Key: "0-9", Desc: "screens"},
		{Key: "/", Desc: "filter"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

// View renders the full-width status bar.
func (s StatusBar) View() string {
	if s.Width <= 0 {
		return ""
	}
	// The bar's own padding, charged before anything is laid out.
	inner := s.Width - styles.StatusBarStyle.GetHorizontalFrameSize()
	if inner < 1 {
		inner = 1
	}

	left := s.leftSegments(inner)

	// Hints keep at most 40% of the bar, so the repository state is never
	// squeezed out by the legend.
	hints := FitHints(s.Hints, inner*2/5)
	hintsW := ansi.StringWidth(hints)

	// Drop left segments from the tail until the hints fit beside them.
	for len(left) > 1 && joinedWidth(left)+hintsW+1 > inner {
		left = left[:len(left)-1]
	}

	body := strings.Join(left, " ")
	bodyW := ansi.StringWidth(body)

	out := hints
	switch {
	case bodyW == 0:
		// No repository state yet: the legend alone.
	case hintsW == 0 || bodyW+hintsW+2 > inner:
		out = Truncate(body, inner)
	default:
		out = body + strings.Repeat(" ", inner-bodyW-hintsW) + hints
	}
	return styles.StatusBarStyle.Width(s.Width).Render(Truncate(out, inner))
}

// joinedWidth is the width of segments joined with single spaces.
func joinedWidth(segs []string) int {
	if len(segs) == 0 {
		return 0
	}
	w := len(segs) - 1 // one space between each pair
	for _, s := range segs {
		w += ansi.StringWidth(s)
	}
	return w
}

// leftSegments builds the repository-state side in priority order: the
// branch always, then the working-tree counts, then where the repo
// lives, then how it talks to GitHub and the AI provider.
func (s StatusBar) leftSegments(inner int) []string {
	g := glyphs.Current()
	d := s.Data

	branch := d.Branch
	if branch == "" {
		branch = "—"
	}
	if tracking := trackingSuffix(d, g); tracking != "" {
		branch += " " + tracking
	}
	branch = Truncate(branch, max(6, inner/3))
	segs := []string{Segment(branch, styles.ChipFgAccent, styles.ChipBgAccent)}

	// Working-tree counts: only what exists.
	var counts []string
	if d.StagedCount > 0 {
		counts = append(counts, Chip(fmt.Sprintf("%s%d", g.Staged, d.StagedCount), styles.TrackAdd, styles.ChipBg))
	}
	if d.UnstagedCount > 0 {
		counts = append(counts, Chip(fmt.Sprintf("%s%d", g.Unstaged, d.UnstagedCount), styles.Warning, styles.ChipBg))
	}
	if d.UntrackedCount > 0 {
		counts = append(counts, Chip(fmt.Sprintf("%s%d", g.Untracked, d.UntrackedCount), styles.Subtext, styles.ChipBg))
	}
	if len(counts) > 0 {
		segs = append(segs, strings.Join(counts, " "))
	}

	if d.RepoPath != "" {
		segs = append(segs, styles.StatusBarMuted.Render(TruncateLeft(d.RepoPath, 34)))
	}
	if d.GhDetected && d.GhOwner != "" && d.GhRepo != "" {
		segs = append(segs, styles.StatusBarMuted.Render(Truncate(d.GhOwner+"/"+d.GhRepo, 28)))
	}
	if d.AIProvider != "" {
		segs = append(segs, styles.MetaStyle.Render(Truncate(d.AIProvider, 24)))
	}
	return segs
}

// trackingSuffix renders "↑2↓1" for the branch's divergence, or "".
func trackingSuffix(d StatusBarData, g glyphs.Set) string {
	up, down := "", ""
	if d.Ahead > 0 {
		up = fmt.Sprintf("%s%d", g.Up, d.Ahead)
	}
	if d.Behind > 0 {
		down = fmt.Sprintf("%s%d", g.Down, d.Behind)
	}
	switch {
	case up != "" && down != "":
		return up + down
	case up != "":
		return up
	default:
		return down
	}
}
