package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/shell"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// longPath is deliberately absurd: a deep monorepo path plus a long name
// is exactly what wrapped the UI on a big repository.
const longPath = "services/platform/internal/very/deep/nested/directory/structure/with_a_really_long_filename_that_never_ends.go"

func frameTestModel(t *testing.T, width, height int) *Model {
	t.Helper()
	gitExec, err := gitpkg.NewNativeExec("")
	if err != nil {
		t.Skip("git not available")
	}
	m := NewModel(gitExec)
	m.ready = true

	// Status: staged + unstaged + untracked, all with overlong paths.
	m.status.UpdateStatus(&models.Status{
		Branch:         "feature/extremely-long-branch-name-that-keeps-going-and-going",
		Upstream:       "origin/feature/extremely-long-branch-name-that-keeps-going-and-going",
		Ahead:          12,
		Behind:         7,
		StagedCount:    2,
		UnstagedCount:  1,
		UntrackedCount: 1,
		Files: []models.FileStatus{
			{Path: longPath, Staged: models.StatusModified},
			{Path: "docs/" + longPath, Staged: models.StatusAdded, Unstaged: models.StatusModified},
			{Path: "tmp/" + longPath, Staged: models.StatusUnmodified, Unstaged: models.StatusModified},
			{Path: "vendor/" + longPath, Staged: models.StatusUntracked, Unstaged: models.StatusUntracked},
		},
	})

	// Log: lane-heavy graph + long subject + many ref badges.
	commits := make([]*models.Commit, 0, 12)
	for i := 0; i < 12; i++ {
		commits = append(commits, &models.Commit{
			Hash:     "abcdef0123456789",
			Parents:  []string{"p1", "p2"},
			Message:  "fix(tui): a commit subject that is far too long for any panel width and will therefore need truncating " + strings.Repeat("x", 40),
			Author:   "octocat",
			RefNames: "HEAD -> feature/extremely-long-branch-name, origin/feature/extremely-long-branch-name, tag:v9.9.9-rc1",
			// Stagger so the graph renderer draws several lanes.
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}
	m.log.UpdateLog(commits)

	m.branches.UpdateBranches([]*models.Branch{
		{Name: "feature/extremely-long-branch-name-that-keeps-going", Upstream: "origin/feature/extremely-long-branch-name-that-keeps-going", IsHead: true, Ahead: 3, LatestMsg: strings.Repeat("long message ", 8)},
		{Name: "main", Upstream: "origin/main"},
	})

	m.prs.UpdatePRs([]*models.PullRequestSummary{
		{Number: 1234, Title: strings.Repeat("a very long pull request title ", 5), Author: "octocat", BaseRef: "dev", HeadRef: "feature/extremely-long-branch-name", ReviewState: "CHANGES_REQUESTED", Labels: []string{"needs-review", "area:tui"}},
	})
	m.issues.UpdateIssues([]*models.Issue{
		{Number: 4321, Title: strings.Repeat("an equally long issue title ", 5), Author: "octocat", Labels: []models.Label{{Name: "bug"}, {Name: "priority:high"}}},
	})
	m.dash.UpdateEntries([]views.DashEntry{
		{Name: "zgit", Path: "/home/someone/Projects/Apps/zgit", Branch: "feature/extremely-long-branch-name", Dirty: 12, Ahead: 4, Behind: 2, Favorite: true, Current: true},
		{Name: strings.Repeat("very-long-repo-name-", 3), Path: "/home/someone/Projects/Apps/" + strings.Repeat("deep/", 6) + "repo", Err: "unreachable"},
	})
	m.actions.UpdateRuns([]*models.WorkflowRun{
		{ID: 1, WorkflowName: strings.Repeat("A very long workflow name ", 3), Branch: "feature/extremely-long-branch-name", HeadSHA: "abcdef0123456789", RunNumber: 9999, Status: "COMPLETED", Conclusion: "SUCCESS"},
	})
	m.notes.UpdateNotes([]*models.Notification{
		{ID: "1", Title: strings.Repeat("long notification title ", 4), Type: "PullRequest", Reason: "review_requested", RepoName: "someone/with-a-long-repository-name", Unread: true},
	})
	m.worktrees.UpdateWorktrees([]*models.Worktree{
		{Path: "/home/someone/Projects/Apps/zgit-worktrees/" + strings.Repeat("feature-branch-", 4), Branch: strings.Repeat("feature/very-long-", 3), Head: "abcdef0123456789"},
	})
	m.stash.UpdateStash([]*models.Stash{
		{Index: 0, Hash: "abcdef0123456789", Message: strings.Repeat("WIP on a very long stash message ", 3)},
	})
	m.refs.UpdateRefs(
		[]string{"v9.9.9-rc1-with-a-very-long-tag-name", strings.Repeat("release-candidate-", 3)},
		[]*models.Remote{{Name: "origin-upstream-with-a-very-long-name", URL: "https://github.com/someone/with-a-really-long-repository-name.git"}},
	)

	m.width = width
	m.height = height
	m.calcLayout()
	return m
}

// TestViewNeverWraps is the regression test for the reported defect: on a
// big repository (long paths, long subjects, many lanes) and after a
// theme switch, rows wrapped and sheared the layout. No rendered line may
// ever exceed the terminal width, in any screen, in either theme.
func TestViewNeverWraps(t *testing.T) {
	// Breakpoints: compact, standard, wide, plus an odd tail width.
	widths := []int{60, 80, 99, 100, 120, 149, 150, 200}
	screens := []shell.ScreenID{shell.Panels, shell.PRs, shell.Issues, shell.Dashboard, shell.Actions, shell.Notes, shell.Worktrees, shell.Stash, shell.Refs}

	// One model, reused: NewModel reads config and probes git, so building
	// it per size would dominate the test's runtime.
	m := frameTestModel(t, 120, 40)

	for _, theme := range []string{"dark", "light"} {
		if theme == "light" {
			styles.SetLight()
		} else {
			styles.SetDark()
		}
		styles.Refresh()

		for _, width := range widths {
			for _, height := range []int{24, 40} {
				m.width = width
				m.height = height
				m.calcLayout()
				for _, screen := range screens {
					m.fullScreenView = screen
					out := m.View()
					if out == "" {
						continue
					}
					for i, line := range strings.Split(out, "\n") {
						if w := lipgloss.Width(line); w > width {
							t.Fatalf("theme=%s screen=%v size=%dx%d line %d is %d cells wide (limit %d): %q",
								theme, screen, width, height, i, w, width, line)
						}
					}
				}
			}
		}
	}
	styles.SetDark()
	styles.Refresh()
}

// TestLayoutFitsWithoutTheSafetyNet renders the header and the content
// column *raw*, without clampFrame in the path. View() clips whatever
// slips through, which would hide a view that is structurally too wide
// (the actual reported defect: rows wrapping and shearing the layout).
func TestLayoutFitsWithoutTheSafetyNet(t *testing.T) {
	widths := []int{60, 80, 99, 100, 120, 149, 150, 200}
	screens := []shell.ScreenID{shell.Panels, shell.PRs, shell.Issues, shell.Dashboard, shell.Actions, shell.Notes, shell.Worktrees, shell.Stash, shell.Refs}

	m := frameTestModel(t, 120, 40)

	for _, theme := range []string{"dark", "light"} {
		if theme == "light" {
			styles.SetLight()
		} else {
			styles.SetDark()
		}
		styles.Refresh()

		for _, width := range widths {
			m.width = width
			m.height = 40
			m.calcLayout()
			for _, screen := range screens {
				m.fullScreenView = screen
				raw := m.renderHeader() + "\n" + m.renderContent()
				for i, line := range strings.Split(raw, "\n") {
					if w := lipgloss.Width(line); w > width {
						t.Errorf("theme=%s screen=%v width=%d: header/content line %d is %d cells: %q",
							theme, screen, width, i, w, line)
					}
				}
			}
		}
	}
	styles.SetDark()
	styles.Refresh()
}

// TestViewNeverWrapsWithSidebar covers the third column: the AI sidebar
// reserves its own width, so it must not push content off-screen.
func TestViewNeverWrapsWithSidebar(t *testing.T) {
	for _, width := range []int{100, 120, 160, 200} {
		m := frameTestModel(t, width, 40)
		m.aiData.Sidebar.State = views.SBSInput
		m.aiData.Sidebar.Messages = []views.ChatMessage{
			{Role: "assistant", Content: strings.Repeat("A long assistant answer that would overflow the sidebar. ", 6)},
			{Role: "user", Content: strings.Repeat("long question ", 10)},
		}
		m.calcLayout()
		out := m.View()
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Fatalf("sidebar size=%d line %d is %d cells wide: %q", width, i, w, line)
			}
		}
	}
}

// TestPanelOccupiesExactWidth pins the container contract: a panel of
// width W renders exactly W columns (border included). Off-by-two here is
// what made every layout two cells too wide.
func TestPanelOccupiesExactWidth(t *testing.T) {
	for _, width := range []int{12, 30, 80, 120} {
		// Content deliberately too wide for the panel.
		content := strings.Repeat("x", 400) + "\n" + strings.Repeat("日本語", 50)
		p := components.NewPanel("Log", width, true)
		p.Height = 8
		p.Content = content
		for i, line := range strings.Split(p.View(), "\n") {
			if w := lipgloss.Width(line); w != width {
				t.Fatalf("panel width %d: line %d is %d cells: %q", width, i, w, line)
			}
		}
	}
}

// TestRowLineChargesForTheFrame proves RowLine can never exceed its width
// even when the row style adds padding.
func TestRowLineChargesForTheFrame(t *testing.T) {
	styles.ListItemActiveStyle = lipgloss.NewStyle().PaddingLeft(2).PaddingRight(3)
	defer styles.Refresh()
	got := components.RowLine(strings.Repeat("wide ", 40), 30, true)
	if w := lipgloss.Width(got); w != 30 {
		t.Fatalf("row is %d cells, want exactly 30: %q", w, got)
	}
}

// TestDialogsFitTerminal measures each modal's raw output (before the
// View() frame guard) so a dialog that outgrows the terminal is a test
// failure rather than a clipped border in production.
func TestDialogsFitTerminal(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 120, 160, 220} {
		m := frameTestModel(t, width, 40)
		m.Update(tea.WindowSizeMsg{Width: width, Height: 40})

		m.commitDlg.Open()
		assertFits(t, "commit", width, m.commitDlg.View(width))
		m.commitDlg.Close()

		m.prCreateDlg.Open("feature/extremely-long-branch-name", "dev")
		assertFits(t, "pr-create", width, m.prCreateDlg.View(width))
		m.prCreateDlg.Close()

		m.configDlg.OpenGitHubToken()
		assertFits(t, "config", width, m.configDlg.View(width))
		m.configDlg.Close()

		m.prMergeDlg.Open(&models.PullRequestSummary{
			Number:    1234,
			Title:     strings.Repeat("an extremely long pull request title ", 4),
			Author:    "octocat",
			HeadRef:   "feature/extremely-long-branch-name-that-keeps-going",
			BaseRef:   "dev",
			Mergeable: "CONFLICTING",
		})
		assertFits(t, "pr-merge", width, m.prMergeDlg.View(width))
		m.prMergeDlg.Close()

		m.palette.Open()
		assertFits(t, "palette", width, m.palette.View(width))
	}
}

// assertFits fails when any line of a rendered block exceeds width cells.
func assertFits(t *testing.T, name string, width int, out string) {
	t.Helper()
	if out == "" {
		return
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("%s at width %d: line %d is %d cells (limit %d): %q", name, width, i, w, width, line)
			return
		}
	}
}
