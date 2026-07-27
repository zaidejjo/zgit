// Package tui provides the Bubble Tea-based terminal UI for zgit.
package tui

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaidejjo/zgit/pkg/core/config"
	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// View IDs for tab switching.
const (
	ViewStatus = iota // 0
	ViewLog           // 1
	ViewBranch        // 2
	ViewPR            // 3
	ViewIssue         // 4
	viewCount
)

// Application modes (overlays and dialogs).
const (
	modeNormal = iota
	modeDiff
	modeCommit
)

// tab labels for the header bar.
var tabNames = []string{"Status", "Log", "Branches", "PRs", "Issues"}

// Model is the root Bubble Tea model for the zgit TUI.
type Model struct {
	// Dependencies
	git  *gitpkg.NativeExec
	gh   *github.CombinedClient // optional, may be nil
	sub  *Subscriber
	msgs chan teaMsg

	// State
	activeView int // currently visible view
	showHelp   bool
	mode       int // normal, diff, commit overlay
	ready      bool
	quitting   bool

	// Sub-models (views)
	status   views.StatusModel
	log      views.LogModel
	branches views.BranchModel
	prs      views.PRListModel
	issues   views.IssueListModel
	helpView views.HelpModel

	// Overlay models
	diffViewer views.DiffModel
	commitDlg  views.CommitModel

	// Bubble Tea components
	help     help.Model
	viewport viewport.Model

	// Terminal dimensions
	width  int
	height int

	// Detected GitHub repo info
	ghOwner    string
	ghRepo     string
	ghDetected bool
}

// NewModel creates the root TUI model.
func NewModel(gitExec *gitpkg.NativeExec) *Model {
	msgs := make(chan teaMsg, 16)

	m := &Model{
		git:        gitExec,
		msgs:       msgs,
		sub:        NewSubscriber(gitExec, msgs),
		status:     views.NewStatusModel(),
		log:        views.NewLogModel(),
		branches:   views.NewBranchModel(),
		prs:        views.NewPRListModel(),
		issues:     views.NewIssueListModel(),
		helpView:   views.HelpModel{},
		diffViewer: views.NewDiffModel(),
		commitDlg:  views.NewCommitModel(),
		help:       help.New(),
		mode:       modeNormal,
	}

	// Try to initialize GitHub client from config
	m.tryInitGitHub()

	// Try to detect GitHub owner/repo from git remotes
	m.tryDetectRepo()

	return m
}

// tryInitGitHub attempts to load a GitHub token from config and create a client.
func (m *Model) tryInitGitHub() {
	cfg, err := config.New()
	if err != nil {
		return
	}
	token := cfg.GetString("github.token")
	if token == "" {
		return
	}
	gh, err := github.NewCombinedClient(token)
	if err != nil {
		return
	}
	m.gh = gh
}

// tryDetectRepo attempts to extract owner/repo from git remotes.
func (m *Model) tryDetectRepo() {
	ctx, cancel := context.WithTimeout(context.Background(), 5e9)
	defer cancel()

	remotes, err := m.git.RemoteList(ctx)
	if err != nil || len(remotes) == 0 {
		return
	}

	// Try origin first, then any remote
	candidates := []*models.Remote{nil}
	for _, r := range remotes {
		if r.Name == "origin" || r.Name == "upstream" {
			candidates = append(candidates, r)
		}
	}
	// Also try the first remote
	if len(remotes) > 0 {
		candidates = append(candidates, remotes[0])
	}

	for _, r := range candidates {
		if r == nil {
			continue
		}
		owner, repo, err := parseGitHubRemote(r.URL)
		if err == nil {
			m.ghOwner = owner
			m.ghRepo = repo
			m.ghDetected = true
			return
		}
	}
}

// parseGitHubRemote extracts owner/repo from a GitHub remote URL.
func parseGitHubRemote(url string) (string, string, error) {
	// Handle formats:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// https://github.com/owner/repo
	url = stringsTrimSuffixGit(url, ".git")
	var path string

	if idx := stringsIndex(url, "github.com/"); idx >= 0 {
		path = url[idx+len("github.com/"):]
	} else if idx := stringsIndex(url, "github.com:"); idx >= 0 {
		path = url[idx+len("github.com:"):]
	} else {
		return "", "", fmt.Errorf("not a GitHub remote: %s", url)
	}

	parts := stringsSplit(path, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("cannot parse owner/repo from %q", url)
}

// Simple string helpers to avoid importing strings in app.go
func stringsTrimSuffixGit(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func stringsIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func stringsSplit(s, sep string) []string {
	var result []string
	for {
		idx := stringsIndex(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	m.sub.Start()
	return tea.Batch(
		listenForMessages(m.msgs),
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If in commit dialog mode, delegate to commit model
	if m.mode == modeCommit {
		return m.updateCommit(msg)
	}

	// If in diff viewer mode, delegate to diff model
	if m.mode == modeDiff {
		return m.updateDiff(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update viewport
		m.viewport = viewport.New(msg.Width, msg.Height-4)
		m.viewport.Style = styles.ContentStyle

		// Update view heights
		m.log.Height = msg.Height - 6
		m.branches.Height = msg.Height - 6
		m.prs.Height = msg.Height - 6
		m.issues.Height = msg.Height - 6
		m.commitDlg.Update(msg)
		m.diffViewer.Update(msg)

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case teaMsg:
		return m.handleEngineMsg(msg)

	default:
		return m, nil
	}
}

// updateCommit delegates updates to the commit dialog model.
func (m *Model) updateCommit(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle "confirm → execute" transition at the app level
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		if m.commitDlg.State == views.CommitConfirming {
			m.commitDlg.State = views.CommitResult
			m.commitDlg.Result = views.CommitResultInfo{
				Success: true, // assume success until goroutine reports otherwise
				Message: m.commitDlg.GetMessage(),
			}
			go m.executeCommit(m.commitDlg.GetMessage())
			return m, nil
		}
	}

	updated, cmd := m.commitDlg.Update(msg)
	m.commitDlg = *updated

	// Close dialog when canceled
	if !m.commitDlg.Active() {
		m.mode = modeNormal
		return m, nil
	}

	// Close result view on keypress (after goroutine finished)
	if m.commitDlg.State == views.CommitResult {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter", "esc":
				shouldRefresh := m.commitDlg.Result.Success && m.commitDlg.Result.Error == ""
				m.commitDlg.Close()
				m.mode = modeNormal
				if shouldRefresh {
					m.sub.Refresh()
				}
				return m, nil
			}
		}
	}

	return m, cmd
}

type commitResultEvent struct {
	Result views.CommitResultInfo
}

// executeCommit runs git commit in a goroutine and sends result via msg channel.
func (m *Model) executeCommit(msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30e9)
	defer cancel()

	opts := gitpkg.CommitOptions{Message: msg}
	err := m.git.Commit(ctx, opts)
	if err != nil {
		m.msgs <- teaMsg{
			view: -1, // special, not a specific view
			data: commitResultEvent{
				Result: views.CommitResultInfo{
					Success: false,
					Error:   err.Error(),
				},
			},
		}
		return
	}

	// Try to get the commit hash from the last commit
	hash := ""
	if commits, logErr := m.git.Log(ctx, gitpkg.LogOptions{Count: 1}); logErr == nil && len(commits) > 0 {
		hash = commits[0].Hash
	}
	m.msgs <- teaMsg{
		view: -1,
		data: commitResultEvent{
			Result: views.CommitResultInfo{
				Success: true,
				Hash:    hash,
				Message: msg,
			},
		},
	}
}

// updateDiff delegates updates to the diff viewer model.
func (m *Model) updateDiff(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.diffViewer.Update(msg)
	m.diffViewer = *updated

	// Escape or Enter closes diff viewer
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "enter":
			if m.diffViewer.Active() {
				m.diffViewer.Clear()
				m.mode = modeNormal
			}
		}
	}

	return m, cmd
}

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return styles.LoadingStyle.Render("Loading...")
	}

	if m.quitting {
		return ""
	}

	// If in an overlay mode, render the overlay on top of the content
	if m.mode == modeCommit {
		tabs := renderTabs(m.activeView, tabNames, m.width)
		content := m.renderContent()
		statusBar := m.renderStatusBar()
		overlay := m.commitDlg.View(m.width)
		return fmt.Sprintf("%s\n%s\n%s\n%s", tabs, content, overlay, statusBar)
	}

	// Render tabs
	tabs := renderTabs(m.activeView, tabNames, m.width)
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	return fmt.Sprintf("%s\n%s\n%s", tabs, content, statusBar)
}

// renderTabs renders the tab bar at the top.
func renderTabs(active int, names []string, width int) string {
	var cells []string
	for i, name := range names {
		if i == active {
			cells = append(cells, styles.ActiveTabStyle.Render(name))
		} else {
			cells = append(cells, styles.InactiveTabStyle.Render(name))
		}
	}
	return styles.AppStyle.Width(width).Render("")
}

// renderContent returns the content area for the active view.
func (m *Model) renderContent() string {
	if m.showHelp {
		return m.helpView.View(m.width)
	}

	// If in diff mode, show diff viewer instead of the active view
	if m.mode == modeDiff && m.diffViewer.Active() {
		return m.diffViewer.View(m.width)
	}

	contentWidth := m.width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	switch m.activeView {
	case ViewStatus:
		return m.status.View(contentWidth)
	case ViewLog:
		return m.log.View(contentWidth)
	case ViewBranch:
		return m.branches.View(contentWidth)
	case ViewPR:
		return m.prs.View(contentWidth)
	case ViewIssue:
		return m.issues.View(contentWidth)
	default:
		return "unknown view"
	}
}

// renderStatusBar shows repo info at the bottom.
func (m *Model) renderStatusBar() string {
	if !m.ready {
		return ""
	}

	branch := "—"
	if m.status.Status != nil {
		branch = m.status.Status.Branch
	}

	left := styles.StatusBarBranchStyle.Render(" " + branch)
	middle := ""
	if m.ghDetected {
		middle = styles.StatusBarInfoStyle.Render(fmt.Sprintf(" %s/%s ", m.ghOwner, m.ghRepo))
	}
	right := styles.StatusBarInfoStyle.Render(" ? help • q quit ")

	bar := styles.StatusBarStyle.
		Width(m.width).
		Render(left + middle + right)

	return bar
}

// handleKeyMsg processes keyboard input.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys that work in all views
	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		m.sub.Stop()
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "r":
		m.sub.Refresh()
		m.refreshGitHubData()
		return m, nil

	case "tab":
		m.activeView = (m.activeView + 1) % viewCount
		m.ensureGHDataLoaded()
		return m, nil

	case "shift+tab":
		m.activeView = (m.activeView - 1 + viewCount) % viewCount
		m.ensureGHDataLoaded()
		return m, nil

	case "1":
		m.activeView = ViewStatus
		return m, nil
	case "2":
		m.activeView = ViewLog
		return m, nil
	case "3":
		m.activeView = ViewBranch
		return m, nil
	case "4":
		m.activeView = ViewPR
		m.ensureGHDataLoaded()
		return m, nil
	case "5":
		m.activeView = ViewIssue
		m.ensureGHDataLoaded()
		return m, nil
	}

	// Delegate to active view
	if m.showHelp {
		return m, nil
	}

	switch m.activeView {
	case ViewStatus:
		return m.handleStatusKeys(key)
	case ViewLog:
		return m.handleLogKeys(key)
	case ViewBranch:
		return m.handleBranchKeys(key)
	case ViewPR:
		return m.handlePRKeys(key)
	case ViewIssue:
		return m.handleIssueKeys(key)
	}

	return m, nil
}

// ensureGHDataLoaded fetches GitHub data if not already loaded for PR/Issue views.
func (m *Model) ensureGHDataLoaded() {
	if !m.ghDetected || m.gh == nil {
		return
	}

	switch m.activeView {
	case ViewPR:
		if m.prs.PRs == nil {
			go m.fetchPRs()
		}
	case ViewIssue:
		if m.issues.Issues == nil {
			go m.fetchIssues()
		}
	}
}

// refreshGitHubData fetches fresh data from GitHub.
func (m *Model) refreshGitHubData() {
	if !m.ghDetected || m.gh == nil {
		return
	}
	go m.fetchPRs()
	go m.fetchIssues()
}

// fetchPRs retrieves open pull requests from GitHub.
func (m *Model) fetchPRs() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: ViewPR, data: prEvent{err: fmt.Errorf("GitHub not authenticated; run 'zgit auth login'")}}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15e9)
	defer cancel()

	opts := github.PRFilter{
		State: "open",
		Sort:  "updated",
		Limit: 50,
	}
	prs, err := m.gh.ListPullRequests(ctx, m.ghOwner, m.ghRepo, opts)
	if err != nil {
		m.msgs <- teaMsg{view: ViewPR, data: prEvent{err: err}}
		return
	}

	// Convert []*models.PullRequestSummary to []*models.PullRequestSummary
	m.msgs <- teaMsg{view: ViewPR, data: prEvent{prs: prs}}
}

// fetchIssues retrieves open issues from GitHub.
func (m *Model) fetchIssues() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: ViewIssue, data: issueEvent{err: fmt.Errorf("GitHub not authenticated; run 'zgit auth login'")}}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15e9)
	defer cancel()

	opts := github.IssuesFilter{
		State: "open",
		Sort:  "updated",
		Limit: 50,
	}
	issues, err := m.gh.ListIssues(ctx, m.ghOwner, m.ghRepo, opts)
	if err != nil {
		m.msgs <- teaMsg{view: ViewIssue, data: issueEvent{err: err}}
		return
	}

	m.msgs <- teaMsg{view: ViewIssue, data: issueEvent{issues: issues}}
}

// handleStatusKeys processes keys for the status view.
func (m *Model) handleStatusKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.status.Status != nil {
			total := len(m.status.Status.Files)
			if total > 0 && m.status.Cursor < total-1 {
				m.status.Cursor++
			}
		}
	case "k", "up":
		if m.status.Cursor > 0 {
			m.status.Cursor--
		}
	case "g", "home":
		m.status.Cursor = 0
	case "G", "end":
		if m.status.Status != nil {
			m.status.Cursor = len(m.status.Status.Files) - 1
			if m.status.Cursor < 0 {
				m.status.Cursor = 0
			}
		}
	case "enter":
		// Open diff viewer for the selected file
		if m.status.Status != nil && m.status.Cursor < len(m.status.Status.Files) {
			m.openFileDiff()
		}
	case "c":
		// Open commit dialog
		m.openCommitDialog()
	}
	return m, nil
}

// openFileDiff opens the diff viewer for the currently selected file.
func (m *Model) openFileDiff() {
	if m.status.Status == nil || m.status.Cursor >= len(m.status.Status.Files) {
		return
	}

	file := m.status.Status.Files[m.status.Cursor]

	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()

	opts := gitpkg.DiffOptions{
		Unified:  true,
		Pathspec: file.Path,
	}
	diff, err := m.git.Diff(ctx, opts)
	if err != nil {
		// Try with different approaches
		opts2 := gitpkg.DiffOptions{
			Unified:  true,
			Pathspec: file.Path,
			Cached:   true,
		}
		diff, err = m.git.Diff(ctx, opts2)
		if err != nil {
			m.status.Error = fmt.Sprintf("diff %s: %v", file.Path, err)
			return
		}
	}

	filePath := file.Path
	if diff != nil && len(diff.Files) > 0 {
		filePath = diff.Files[0].NewPath
		if filePath == "" {
			filePath = diff.Files[0].OldPath
		}
		unifiedDiff := ""
		if len(diff.Files) > 0 {
			unifiedDiff = diff.Files[0].UnifiedDiff
		}
		m.diffViewer.SetDiff(filePath, unifiedDiff, diff.TotalAdds, diff.TotalDeletes)
	} else {
		m.diffViewer.SetDiff(file.Path, "(no changes)", 0, 0)
	}

	m.mode = modeDiff
}

// openCommitDialog opens the interactive commit dialog.
func (m *Model) openCommitDialog() {
	m.commitDlg.Open()
	m.mode = modeCommit
}

// handleLogKeys processes keys for the log view.
func (m *Model) handleLogKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if len(m.log.Commits) > 0 && m.log.Cursor < len(m.log.Commits)-1 {
			m.log.Cursor++
		}
	case "k", "up":
		if m.log.Cursor > 0 {
			m.log.Cursor--
		}
	case "g", "home":
		m.log.Cursor = 0
	case "G", "end":
		m.log.Cursor = len(m.log.Commits) - 1
		if m.log.Cursor < 0 {
			m.log.Cursor = 0
		}
	case "pgdown", "ctrl+f":
		m.log.Cursor += m.log.Height
		if m.log.Cursor >= len(m.log.Commits) {
			m.log.Cursor = len(m.log.Commits) - 1
		}
	case "pgup", "ctrl+b":
		m.log.Cursor -= m.log.Height
		if m.log.Cursor < 0 {
			m.log.Cursor = 0
		}
	}
	return m, nil
}

// handleBranchKeys processes keys for the branch view.
func (m *Model) handleBranchKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if len(m.branches.Branches) > 0 && m.branches.Cursor < len(m.branches.Branches)-1 {
			m.branches.Cursor++
		}
	case "k", "up":
		if m.branches.Cursor > 0 {
			m.branches.Cursor--
		}
	case "g", "home":
		m.branches.Cursor = 0
	case "G", "end":
		m.branches.Cursor = len(m.branches.Branches) - 1
		if m.branches.Cursor < 0 {
			m.branches.Cursor = 0
		}
	case "c", "enter":
		if m.branches.Cursor >= 0 && m.branches.Cursor < len(m.branches.Branches) {
			b := m.branches.Branches[m.branches.Cursor]
			if err := m.git.Checkout(context.Background(), b.Name); err != nil {
				m.branches.Error = fmt.Sprintf("checkout %s: %v", b.Name, err)
			} else {
				m.sub.Refresh()
			}
		}
	}
	return m, nil
}

// handlePRKeys processes keys for the PR list view.
func (m *Model) handlePRKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if len(m.prs.PRs) > 0 && m.prs.Cursor < len(m.prs.PRs)-1 {
			m.prs.Cursor++
		}
	case "k", "up":
		if m.prs.Cursor > 0 {
			m.prs.Cursor--
		}
	case "g", "home":
		m.prs.Cursor = 0
	case "G", "end":
		m.prs.Cursor = len(m.prs.PRs) - 1
		if m.prs.Cursor < 0 {
			m.prs.Cursor = 0
		}
	case "pgdown", "ctrl+f":
		m.prs.Cursor += m.prs.Height
		if m.prs.Cursor >= len(m.prs.PRs) {
			m.prs.Cursor = len(m.prs.PRs) - 1
		}
	case "pgup", "ctrl+b":
		m.prs.Cursor -= m.prs.Height
		if m.prs.Cursor < 0 {
			m.prs.Cursor = 0
		}
	}
	return m, nil
}

// handleIssueKeys processes keys for the issue list view.
func (m *Model) handleIssueKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if len(m.issues.Issues) > 0 && m.issues.Cursor < len(m.issues.Issues)-1 {
			m.issues.Cursor++
		}
	case "k", "up":
		if m.issues.Cursor > 0 {
			m.issues.Cursor--
		}
	case "g", "home":
		m.issues.Cursor = 0
	case "G", "end":
		m.issues.Cursor = len(m.issues.Issues) - 1
		if m.issues.Cursor < 0 {
			m.issues.Cursor = 0
		}
	case "pgdown", "ctrl+f":
		m.issues.Cursor += m.issues.Height
		if m.issues.Cursor >= len(m.issues.Issues) {
			m.issues.Cursor = len(m.issues.Issues) - 1
		}
	case "pgup", "ctrl+b":
		m.issues.Cursor -= m.issues.Height
		if m.issues.Cursor < 0 {
			m.issues.Cursor = 0
		}
	}
	return m, nil
}

// --- Event types for GitHub data ---

type prEvent struct {
	prs []*models.PullRequestSummary
	err error
}

type issueEvent struct {
	issues []*models.Issue
	err    error
}

// handleEngineMsg processes messages from the background subscriber.
func (m *Model) handleEngineMsg(msg teaMsg) (tea.Model, tea.Cmd) {
	// Check for commit result event first (view == -1)
	if msg.view == -1 {
		if evt, ok := msg.data.(commitResultEvent); ok {
			m.commitDlg.Result = evt.Result
			m.commitDlg.State = views.CommitResult
		}
		return m, nil
	}

	switch msg.view {
	case ViewStatus:
		if evt, ok := msg.data.(statusEvent); ok {
			if evt.err != nil {
				m.status.Error = evt.err.Error()
			} else if evt.status != nil {
				m.status.UpdateStatus(evt.status)
			}
		}
	case ViewLog:
		if evt, ok := msg.data.(logEvent); ok {
			if evt.err != nil {
				m.log.Error = evt.err.Error()
			} else if evt.commits != nil {
				m.log.UpdateLog(evt.commits)
			}
		}
	case ViewBranch:
		if evt, ok := msg.data.(branchEvent); ok {
			if evt.err != nil {
				m.branches.Error = evt.err.Error()
			} else if evt.branches != nil {
				m.branches.UpdateBranches(evt.branches)
			}
		}
	case ViewPR:
		if evt, ok := msg.data.(prEvent); ok {
			if evt.err != nil {
				m.prs.Error = evt.err.Error()
			} else if evt.prs != nil {
				m.prs.UpdatePRs(evt.prs)
			}
		}
	case ViewIssue:
		if evt, ok := msg.data.(issueEvent); ok {
			if evt.err != nil {
				m.issues.Error = evt.err.Error()
			} else if evt.issues != nil {
				m.issues.UpdateIssues(evt.issues)
			}
		}
	}
	return m, nil
}

// listenForMessages creates a command that polls the message channel.
func listenForMessages(msgs <-chan teaMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-msgs
		if !ok {
			return nil
		}
		return msg
	}
}

// Run starts the TUI application.
func Run(gitExec *gitpkg.NativeExec) error {
	m := NewModel(gitExec)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		return err
	}
	return nil
}
