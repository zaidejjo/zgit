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
	GhUser         string // authenticated username, e.g. "octocat"
	AIProvider     string // e.g. "openai:gpt-4o" or empty
	RepoPath       string // absolute path to repository, shown on hover/truncated
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

	// Repository path (truncated on left if too long)
	repoPath := ""
	if s.Data.RepoPath != "" {
		repoShort := s.Data.RepoPath
		maxPathLen := 40
		if len(repoShort) > maxPathLen {
			// Truncate from left to fit
			repoShort = "…" + repoShort[len(repoShort)-maxPathLen+1:]
		}
		repoPath = styles.StatusBarInfoStyle.Render(" " + repoShort + " ")
	}

	left := styles.StatusBarBranchStyle.Render(" "+branch) +
		styles.SubtitleStyle.Render(track) + repoPath

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

	// Right: AI provider + user + repo + help hint
	right := ""
	if s.Data.AIProvider != "" {
		right = styles.StatusBarInfoStyle.Render(" " + s.Data.AIProvider + " ")
	}
	if s.Data.GhUser != "" {
		right += styles.StatusBarInfoStyle.Render(fmt.Sprintf(" @%s ", s.Data.GhUser))
	}
	if s.Data.GhDetected {
		right += styles.StatusBarInfoStyle.Render(fmt.Sprintf(" %s/%s ", s.Data.GhOwner, s.Data.GhRepo))
	}
	right += styles.StatusBarInfoStyle.Render(" ?help qquit ")

	return styles.StatusBarStyle.
		Width(s.Width).
		Render(left + " " + middle + " " + right)
}
