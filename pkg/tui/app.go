// Package tui provides the Bubble Tea-based terminal UI for zgit.
package tui

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/config"
	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// Panel identifiers for focus cycling.
const (
	PanelStatus = iota // 0
	PanelLog           // 1
	PanelBranch        // 2
	panelCount
)

// Full-screen view identifiers (override panel layout).
// Values 3+ to avoid overlap with subscriber event IDs (0=status,1=log,2=branches).
const (
	fsNone  = -1 // normal panel layout
	fsPR    = 3  // matches old ViewPR for subscriber compatibility
	fsIssue = 4  // matches old ViewIssue
)

// Application modes (overlays and dialogs).
const (
	modeNormal = iota
	modeDiff
	modeCommit
)

// Model is the root Bubble Tea model for the zgit TUI.
type Model struct {
	// Dependencies
	git  *gitpkg.NativeExec
	gh   *github.CombinedClient // optional, may be nil
	sub  *Subscriber
	msgs chan teaMsg

	// State
	focusedPanel   int // which panel has cursor focus (0-2)
	fullScreenView int // -1 = panels, fsPR, fsIssue
	showHelp       bool
	mode           int // normal, diff, commit overlay
	ready          bool
	quitting       bool

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

	// Layout dimensions (computed on resize)
	leftWidth     int
	rightWidth    int
	contentHeight int

	// Detected GitHub repo info
	ghOwner    string
	ghRepo     string
	ghDetected bool
}

// NewModel creates the root TUI model.
func NewModel(gitExec *gitpkg.NativeExec) *Model {
	msgs := make(chan teaMsg, 16)

	m := &Model{
		git:            gitExec,
		msgs:           msgs,
		sub:            NewSubscriber(gitExec, msgs),
		status:         views.NewStatusModel(),
		log:            views.NewLogModel(),
		branches:       views.NewBranchModel(),
		prs:            views.NewPRListModel(),
		issues:         views.NewIssueListModel(),
		helpView:       views.HelpModel{},
		diffViewer:     views.NewDiffModel(),
		commitDlg:      views.NewCommitModel(),
		help:           help.New(),
		mode:           modeNormal,
		focusedPanel:   PanelStatus,
		fullScreenView: fsNone,
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

// Simple string helpers.
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

// --- Layout calculations ---

func (m *Model) calcLayout() {
	statusBarH := 1
	m.contentHeight = m.height - statusBarH
	if m.contentHeight < 4 {
		m.contentHeight = 4
	}

	m.leftWidth = int(float64(m.width)*0.4 + 0.5)
	m.rightWidth = m.width - m.leftWidth

	// Minimum widths
	if m.leftWidth < 24 {
		m.leftWidth = 24
		m.rightWidth = m.width - m.leftWidth
	}
	if m.rightWidth < 30 {
		m.rightWidth = 30
		m.leftWidth = m.width - m.rightWidth
	}
	if m.leftWidth < 24 {
		m.leftWidth = 24
		m.rightWidth = m.width - m.leftWidth
	}
	if m.leftWidth < 4 {
		m.leftWidth = 4
	}
	if m.rightWidth < 4 {
		m.rightWidth = 4
	}

	// Panel heights for right side (log top, branches bottom)
	// Each panel loses 2 for border + 1 for title = 3 lines of overhead
	logPanelH := int(float64(m.contentHeight)*0.55 + 0.5)
	branchPanelH := m.contentHeight - logPanelH

	// Set view scroll heights (inner height minus title line)
	m.log.Height = logPanelH - 3
	if m.log.Height < 1 {
		m.log.Height = 1
	}
	m.branches.Height = branchPanelH - 3
	if m.branches.Height < 1 {
		m.branches.Height = 1
	}
	m.prs.Height = m.contentHeight - 4
	if m.prs.Height < 1 {
		m.prs.Height = 1
	}
	m.issues.Height = m.contentHeight - 4
	if m.issues.Height < 1 {
		m.issues.Height = 1
	}
}

// --- Update ---

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
		m.calcLayout()

		// Update viewport and overlays
		m.viewport = viewport.New(msg.Width, m.height-4)
		m.viewport.Style = styles.ContentStyle
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

// --- View ---

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return styles.LoadingStyle.Render("Loading...")
	}

	if m.quitting {
		return ""
	}

	// Determine which content to show
	content := m.renderContent()

	// Overlays render on top of content
	if m.mode == modeCommit {
		overlay := m.commitDlg.View(m.width)
		statusBar := m.renderStatusBar()
		return fmt.Sprintf("%s\n%s\n%s", content, overlay, statusBar)
	}

	statusBar := m.renderStatusBar()
	return fmt.Sprintf("%s\n%s", content, statusBar)
}

// renderContent returns either the panel layout or a full-screen view.
func (m *Model) renderContent() string {
	if m.showHelp {
		return m.helpView.View(m.width)
	}

	// If in diff mode, show diff viewer full-screen
	if m.mode == modeDiff && m.diffViewer.Active() {
		return m.diffViewer.View(m.width)
	}

	// Full-screen views (PRs, Issues)
	switch m.fullScreenView {
	case fsPR:
		return m.prs.View(m.width - 2)
	case fsIssue:
		return m.issues.View(m.width - 2)
	}

	// Panel layout: Status | (Log / Branches)
	return m.renderPanels()
}

// renderPanels builds the horizontal split layout.
func (m *Model) renderPanels() string {
	leftInnerW := m.leftWidth - 2 // minus panel border
	rightInnerW := m.rightWidth - 2

	// Compute panel heights
	logPanelH := int(float64(m.contentHeight)*0.55 + 0.5)
	branchPanelH := m.contentHeight - logPanelH
	// Status panel fills full height of left column
	statusPanelH := m.contentHeight

	// Render view content into inner widths
	statusContent := m.status.View(leftInnerW)
	logContent := m.log.View(rightInnerW)
	branchContent := m.branches.View(rightInnerW)

	// Build panels with fixed heights
	statusPanel := components.NewPanel("Status", m.leftWidth, m.focusedPanel == PanelStatus)
	statusPanel.Height = statusPanelH
	statusPanel.Content = statusContent

	logPanel := components.NewPanel("Log", m.rightWidth, m.focusedPanel == PanelLog)
	logPanel.Height = logPanelH
	logPanel.Content = logContent

	branchPanel := components.NewPanel("Branches", m.rightWidth, m.focusedPanel == PanelBranch)
	branchPanel.Height = branchPanelH
	branchPanel.Content = branchContent

	// Assemble: left column full-height, right column split vertically
	rightCol := lipgloss.JoinVertical(
		lipgloss.Top,
		logPanel.View(),
		branchPanel.View(),
	)

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statusPanel.View(),
		rightCol,
	)

	return mainContent
}

// renderStatusBar builds the bottom bar.
func (m *Model) renderStatusBar() string {
	if !m.ready {
		return ""
	}

	data := components.StatusBarData{
		Branch:     "—",
		GhOwner:    m.ghOwner,
		GhRepo:     m.ghRepo,
		GhDetected: m.ghDetected,
	}

	if m.status.Status != nil {
		data.Branch = m.status.Status.Branch
		data.Ahead = m.status.Status.Ahead
		data.Behind = m.status.Status.Behind
		data.StagedCount = m.status.Status.StagedCount
		data.UnstagedCount = m.status.Status.UnstagedCount
		data.UntrackedCount = len(m.status.Status.UntrackedFiles())
	}

	bar := components.NewStatusBar(m.width, data)
	return bar.View()
}

// --- Key handling ---

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
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

	// Tab cycles focus: Status → Log → Branches → Status
	case "tab":
		if m.fullScreenView != fsNone {
			// In full-screen view, Tab goes back to panels
			m.fullScreenView = fsNone
		} else {
			m.focusedPanel = (m.focusedPanel + 1) % panelCount
		}
		return m, nil

	case "shift+tab":
		if m.fullScreenView != fsNone {
			m.fullScreenView = fsNone
		} else {
			m.focusedPanel = (m.focusedPanel - 1 + panelCount) % panelCount
		}
		return m, nil

	// Full-screen view switches
	case "3":
		m.fullScreenView = fsPR
		m.ensureGHDataLoaded()
		return m, nil
	case "4":
		m.fullScreenView = fsIssue
		m.ensureGHDataLoaded()
		return m, nil
	}

	// If help is open, no other keys
	if m.showHelp {
		return m, nil
	}

	// If in full-screen view (PR/Issue), delegate to those handlers
	if m.fullScreenView == fsPR {
		return m.handlePRKeys(key)
	}
	if m.fullScreenView == fsIssue {
		return m.handleIssueKeys(key)
	}

	// Delegate to focused panel
	switch m.focusedPanel {
	case PanelStatus:
		return m.handleStatusKeys(key)
	case PanelLog:
		return m.handleLogKeys(key)
	case PanelBranch:
		return m.handleBranchKeys(key)
	}

	return m, nil
}

// --- GitHub data loading ---

func (m *Model) ensureGHDataLoaded() {
	if !m.ghDetected || m.gh == nil {
		return
	}

	switch m.fullScreenView {
	case 3: // PRs
		if m.prs.PRs == nil {
			go m.fetchPRs()
		}
	case 4: // Issues
		if m.issues.Issues == nil {
			go m.fetchIssues()
		}
	}
}

func (m *Model) refreshGitHubData() {
	if !m.ghDetected || m.gh == nil {
		return
	}
	go m.fetchPRs()
	go m.fetchIssues()
}

func (m *Model) fetchPRs() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: 3, data: prEvent{err: fmt.Errorf("GitHub not authenticated; run 'zgit auth login'")}}
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
		m.msgs <- teaMsg{view: 3, data: prEvent{err: err}}
		return
	}

	m.msgs <- teaMsg{view: 3, data: prEvent{prs: prs}}
}

func (m *Model) fetchIssues() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: 4, data: issueEvent{err: fmt.Errorf("GitHub not authenticated; run 'zgit auth login'")}}
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
		m.msgs <- teaMsg{view: 4, data: issueEvent{err: err}}
		return
	}

	m.msgs <- teaMsg{view: 4, data: issueEvent{issues: issues}}
}

// --- View-specific key handlers ---

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
		if m.status.Status != nil && m.status.Cursor < len(m.status.Status.Files) {
			m.openFileDiff()
		}
	case "c":
		m.openCommitDialog()
	}
	return m, nil
}

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

// --- Diff & Commit ---

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

func (m *Model) openCommitDialog() {
	m.commitDlg.Open()
	m.mode = modeCommit
}

func (m *Model) updateCommit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		if m.commitDlg.State == views.CommitConfirming {
			m.commitDlg.State = views.CommitResult
			m.commitDlg.Result = views.CommitResultInfo{
				Success: true,
				Message: m.commitDlg.GetMessage(),
			}
			go m.executeCommit(m.commitDlg.GetMessage())
			return m, nil
		}
	}

	updated, cmd := m.commitDlg.Update(msg)
	m.commitDlg = *updated

	if !m.commitDlg.Active() {
		m.mode = modeNormal
		return m, nil
	}

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

func (m *Model) executeCommit(msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30e9)
	defer cancel()

	opts := gitpkg.CommitOptions{Message: msg}
	err := m.git.Commit(ctx, opts)
	if err != nil {
		m.msgs <- teaMsg{
			view: -1,
			data: commitResultEvent{
				Result: views.CommitResultInfo{
					Success: false,
					Error:   err.Error(),
				},
			},
		}
		return
	}

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

func (m *Model) updateDiff(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.diffViewer.Update(msg)
	m.diffViewer = *updated

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

// --- Event types ---

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
	if msg.view == -1 {
		if evt, ok := msg.data.(commitResultEvent); ok {
			m.commitDlg.Result = evt.Result
			m.commitDlg.State = views.CommitResult
		}
		return m, nil
	}

	switch msg.view {
	case 0: // status
		if evt, ok := msg.data.(statusEvent); ok {
			if evt.err != nil {
				m.status.Error = evt.err.Error()
			} else if evt.status != nil {
				m.status.UpdateStatus(evt.status)
			}
		}
	case 1: // log
		if evt, ok := msg.data.(logEvent); ok {
			if evt.err != nil {
				m.log.Error = evt.err.Error()
			} else if evt.commits != nil {
				m.log.UpdateLog(evt.commits)
			}
		}
	case 2: // branches
		if evt, ok := msg.data.(branchEvent); ok {
			if evt.err != nil {
				m.branches.Error = evt.err.Error()
			} else if evt.branches != nil {
				m.branches.UpdateBranches(evt.branches)
			}
		}
	case 3: // PRs
		if evt, ok := msg.data.(prEvent); ok {
			if evt.err != nil {
				m.prs.Error = evt.err.Error()
			} else if evt.prs != nil {
				m.prs.UpdatePRs(evt.prs)
			}
		}
	case 4: // Issues
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
