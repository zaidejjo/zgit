package components

import (
	"fmt"

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
}

// StatusBar renders the bottom bar with repo info.
type StatusBar struct {
	Width int
	Data  StatusBarData
}

// NewStatusBar creates a status bar component.
func NewStatusBar(width int, data StatusBarData) StatusBar {
	return StatusBar{
		Width: width,
		Data:  data,
	}
}

// View renders the full-width status bar.
func (s StatusBar) View() string {
	branch := s.Data.Branch
	if branch == "" {
		branch = "—"
	}

	// Left: branch name with tracking
	track := ""
	if s.Data.Ahead > 0 || s.Data.Behind > 0 {
		if s.Data.Ahead > 0 {
			track += fmt.Sprintf(" +%d", s.Data.Ahead)
		}
		if s.Data.Behind > 0 {
			track += fmt.Sprintf(" -%d", s.Data.Behind)
		}
	}

	left := styles.StatusBarBranchStyle.Render(" "+branch) +
		styles.SubtitleStyle.Render(track)

	// Middle: file counts
	middle := ""
	if s.Data.StagedCount > 0 {
		middle += " " + styles.StatusStagedStyle.Render(fmt.Sprintf("+%d", s.Data.StagedCount))
	}
	if s.Data.UnstagedCount > 0 {
		middle += " " + styles.StatusUnstagedStyle.Render(fmt.Sprintf("~%d", s.Data.UnstagedCount))
	}
	if s.Data.UntrackedCount > 0 {
		middle += " " + styles.StatusUntrackedStyle.Render(fmt.Sprintf("?%d", s.Data.UntrackedCount))
	}

	// Right: repo + help hint
	right := ""
	if s.Data.GhDetected {
		right = styles.StatusBarInfoStyle.Render(fmt.Sprintf(" %s/%s ", s.Data.GhOwner, s.Data.GhRepo))
	}
	right += styles.StatusBarInfoStyle.Render(" ?help qquit ")

	return styles.StatusBarStyle.
		Width(s.Width).
		Render(left + " " + middle + " " + right)
}
