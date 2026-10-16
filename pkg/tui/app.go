// Package tui provides the Bubble Tea-based terminal UI for zgit.
package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

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

// AppVersion is the release version shown in the TUI header.
// Set from cmd/zgit/main.go's ldflag-injected Version at startup.
var AppVersion = "dev"

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
	fsNone      = -1 // normal panel layout
	fsPR        = 3  // matches old ViewPR for subscriber compatibility
	fsIssue     = 4  // matches old ViewIssue
	fsDashboard = 5
	fsActions   = 6
	fsNotes     = 7
)

// Application modes (overlays and dialogs).
const (
	modeNormal = iota
	modeDiff
	modeCommit
	modeConfig
	modePRCreate
	modePRMerge
)

// Model is the root Bubble Tea model for the zgit TUI.
type Model struct {
	// Dependencies
	git  *gitpkg.NativeExec
	gh   *github.CombinedClient // optional, may be nil
	sub  *Subscriber
	msgs chan teaMsg

	// State
	focusedPanel    int // which panel has cursor focus (0-2)
	fullScreenView  int // -1 = panels, fsPR, fsIssue
	showHelp        bool
	mode            int // normal, diff, commit overlay
	ready           bool
	quitting        bool
	pushAfterCommit bool // if true, push after commit succeeds

	// Sub-models (views)
	status   views.StatusModel
	log      views.LogModel
	branches views.BranchModel
	prs      views.PRListModel
	issues   views.IssueListModel
	dash     views.DashModel
	actions  views.ActionsModel
	notes    views.NotesModel
	helpView views.HelpModel

	// Overlay models
	diffViewer  views.DiffModel
	commitDlg   views.CommitModel
	configDlg   views.ConfigModel
	palette     views.PaletteModel
	prCreateDlg views.PRCreateModel
	prMergeDlg  views.PRMergeModel

	// Bubble Tea components
	help     help.Model
	viewport viewport.Model

	// Terminal dimensions
	width  int
	height int

	// Layout dimensions (computed on resize)
	leftWidth     int
	rightWidth    int
	sidebarWidth  int
	contentHeight int

	// Detected GitHub repo info
	ghOwner    string
	ghRepo     string
	ghDetected bool
	ghUser     string // authenticated GitHub username

	// AI integration
	aiData aiState
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
		dash:           views.NewDashModel(),
		actions:        views.NewActionsModel(),
		notes:          views.NewNotesModel(),
		helpView:       views.HelpModel{},
		diffViewer:     views.NewDiffModel(),
		commitDlg:      views.NewCommitModel(),
		configDlg:      views.NewConfigModel(),
		palette:        views.NewPaletteModel(),
		prCreateDlg:    views.NewPRCreateModel(),
		prMergeDlg:     views.NewPRMergeModel(),
		help:           help.New(),
		mode:           modeNormal,
		focusedPanel:   PanelStatus,
		fullScreenView: fsNone,
	}

	// Try to initialize GitHub client from config
	m.tryInitGitHub()

	// Initialize AI state
	m.initAI()

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

	// Load cached user from config
	m.ghUser = cfg.GetString("github.user")

	// Try to fetch user if not cached
	if m.ghUser == "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10e9)
			defer cancel()
			user, err := gh.GetAuthenticatedUser(ctx)
			if err == nil && user != nil {
				cfg.Set("github.user", user.Login)
				_ = cfg.Save()
				m.ghUser = user.Login
			}
		}()
	}
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

// initialRefreshCmd triggers the first data fetch after the event loop is ready.
func (m *Model) initialRefreshCmd() tea.Msg {
	m.sub.Refresh()
	return nil
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	m.sub.Start()
	return tea.Batch(
		listenForMessages(m.msgs),
		m.initialRefreshCmd,
	)
}

// --- Layout calculations ---

func (m *Model) calcLayout() {
	statusBarH := 1
	headerH := 1
	m.contentHeight = m.height - statusBarH - headerH
	if m.contentHeight < 4 {
		m.contentHeight = 4
	}

	// Reserve sidebar width if active
	m.sidebarWidth = 0
	if m.aiData.Sidebar.Active() {
		m.sidebarWidth = 42 // matches Width(42) in sidebar View (includes border+padding)
		m.aiData.Sidebar.Width = 42
		m.aiData.Sidebar.Height = m.contentHeight
	}

	availW := m.width - m.sidebarWidth
	if availW < 54 {
		availW = 54 // minimum viable panel width
		m.sidebarWidth = m.width - availW
		if m.sidebarWidth < 0 {
			m.sidebarWidth = 0
		}
	}

	m.leftWidth = int(float64(availW)*0.4 + 0.5)
	m.rightWidth = availW - m.leftWidth

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
	// Dialog mode dispatch
	switch m.mode {
	case modeCommit:
		return m.updateCommit(msg)
	case modeDiff:
		return m.updateDiff(msg)
	case modeConfig:
		return m.updateConfig(msg)
	case modePRCreate:
		return m.updatePRCreate(msg)
	case modePRMerge:
		return m.updatePRMerge(msg)
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
		// Route to command palette if active (overrides everything)
		if m.palette.Active() {
			return m.updatePalette(msg)
		}

		// Route to AI sidebar if active (handles input + commands)
		if m.aiData.Sidebar.Active() {
			updated, cmd := m.handleAIKey(msg.String())
			return updated, cmd
		}
		return m.handleKeyMsg(msg)

	case teaMsg:
		updated, cmd := m.handleEngineMsg(msg)
		// Reinstall listener for next subscriber message
		return updated, tea.Batch(cmd, listenForMessages(m.msgs))

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

	// Base layout: header + content + status bar
	content := m.renderHeader() + "\n" + m.renderContent()
	statusBar := m.renderStatusBar()
	fullLayout := content
	if statusBar != "" {
		fullLayout = fmt.Sprintf("%s\n%s", content, statusBar)
	}

	// Pad fullLayout to exactly m.height lines so overlay centering is pixel-perfect
	baseLines := strings.Split(fullLayout, "\n")
	if len(baseLines) < m.height {
		padding := make([]string, m.height-len(baseLines))
		for i := range padding {
			padding[i] = strings.Repeat(" ", m.width)
		}
		fullLayout = fullLayout + "\n" + strings.Join(padding, "\n")
	}

	// Floating modal overlays — centered on top of everything, no layout shift
	switch {
	case m.mode == modeCommit:
		overlay := m.commitDlg.View(m.width)
		if overlay != "" {
			return overlayCenter(fullLayout, overlay, m.width, m.height)
		}
	case m.mode == modeConfig:
		overlay := m.configDlg.View(m.width)
		if overlay != "" {
			return overlayCenter(fullLayout, overlay, m.width, m.height)
		}
	case m.mode == modePRCreate:
		overlay := m.prCreateDlg.View(m.width)
		if overlay != "" {
			return overlayCenter(fullLayout, overlay, m.width, m.height)
		}
	case m.mode == modePRMerge:
		overlay := m.prMergeDlg.View(m.width)
		if overlay != "" {
			return overlayCenter(fullLayout, overlay, m.width, m.height)
		}
	}

	if m.palette.Active() {
		overlay := m.palette.View(m.width)
		if overlay != "" {
			return overlayCenter(fullLayout, overlay, m.width, m.height)
		}
	}

	return fullLayout
}

// renderHeader builds the Dale-style top tab bar:
// [zgit] Status Log Branches │ 3 PRs  4 Issues │  @user  main  vX.Y.Z
// The active view (focused panel or fullscreen) is highlighted.
func (m *Model) renderHeader() string {
	tab := func(label string, active bool) string {
		if active {
			return styles.ActiveTabStyle.Render(label)
		}
		return styles.InactiveTabStyle.Render(label)
	}

	var tabs []string
	tabs = append(tabs, styles.HeaderLogoStyle.Render(" zgit "))
	panelActive := m.fullScreenView == fsNone
	tabs = append(tabs, tab("Status", panelActive && m.focusedPanel == PanelStatus))
	tabs = append(tabs, tab("Log", panelActive && m.focusedPanel == PanelLog))
	tabs = append(tabs, tab("Branches", panelActive && m.focusedPanel == PanelBranch))
	tabs = append(tabs, tab("3 PRs", m.fullScreenView == fsPR))
	tabs = append(tabs, tab("4 Issues", m.fullScreenView == fsIssue))
	// Compact labels on narrow terminals so the strip never wraps.
	reposLabel, actionsLabel := "5 Repos", "6 Actions"
	if m.width > 0 && m.width < 110 {
		reposLabel, actionsLabel = "5R", "6A"
	}
	tabs = append(tabs, tab(reposLabel, m.fullScreenView == fsDashboard))
	tabs = append(tabs, tab(actionsLabel, m.fullScreenView == fsActions))
	notesLabel := "7 Notes"
	if m.notes.Loaded && m.notes.UnreadCount() > 0 {
		notesLabel = fmt.Sprintf("7 Notes (%d)", m.notes.UnreadCount())
	}
	tabs = append(tabs, tab(notesLabel, m.fullScreenView == fsNotes))
	left := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)

	var meta []string
	if m.ghUser != "" {
		meta = append(meta, styles.HeaderMetaStyle.Render("@"+m.ghUser))
	}
	branch := m.currentBranch()
	if branch != "" {
		meta = append(meta, styles.HeaderBranchStyle.Render(branch))
	}
	meta = append(meta, styles.HeaderMetaStyle.Render("v"+AppVersion))
	right := lipgloss.JoinHorizontal(lipgloss.Center, meta...)

	// Pad the middle so meta hugs the right edge; truncate tabs on narrow terms.
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
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
	case fsDashboard:
		return m.dash.View(m.width - 2)
	case fsActions:
		return m.actions.View(m.width - 2)
	case fsNotes:
		return m.notes.View(m.width - 2)
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

	// Append AI sidebar as rightmost column when active
	if m.aiData.Sidebar.Active() {
		sidebarView := m.aiData.Sidebar.View()
		if sidebarView != "" {
			// Align sidebar height with panels (exactly contentHeight lines)
			sideLines := strings.Split(sidebarView, "\n")
			if len(sideLines) > m.contentHeight {
				sideLines = sideLines[:m.contentHeight]
			} else {
				blank := strings.Repeat(" ", m.sidebarWidth)
				for len(sideLines) < m.contentHeight {
					sideLines = append(sideLines, blank)
				}
			}
			sidebarView = strings.Join(sideLines, "\n")
			mainContent = lipgloss.JoinHorizontal(
				lipgloss.Top,
				mainContent,
				sidebarView,
			)
		}
	}

	return mainContent
}

// renderStatusBar builds the bottom bar.
func (m *Model) renderStatusBar() string {
	if !m.ready {
		return ""
	}

	repoPath := ""
	if m.git != nil {
		repoPath = m.git.Path()
	}
	data := components.StatusBarData{
		Branch:     "—",
		GhOwner:    m.ghOwner,
		GhRepo:     m.ghRepo,
		GhDetected: m.ghDetected,
		GhUser:     m.ghUser,
		AIProvider: m.aiProviderLabel(),
		RepoPath:   repoPath,
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
		if m.showHelp {
			m.showHelp = false
			m.helpView.Input = ""
			m.helpView.Cursor = 0
			m.helpView.ShowInput = false
		} else {
			m.showHelp = true
			m.helpView.Input = ""
			m.helpView.Cursor = 0
			m.helpView.ShowInput = false
		}
		return m, nil

	case "R":
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

	// Command palette (Ctrl+P)
	case "ctrl+p":
		if m.mode == modeNormal {
			m.palette.Open()
		}
		return m, nil

	// Config: GitHub PAT or AI settings (Ctrl+T)
	case "ctrl+t":
		if m.mode == modeNormal {
			m.configDlg.OpenGitHubToken()
			m.mode = modeConfig
		}
		return m, nil

	// AI: quick ask sidebar (Ctrl+G)
	case "ctrl+g":
		if m.mode == modeNormal && m.fullScreenView == fsNone {
			if m.aiData.Sidebar.Active() {
				m.aiData.Sidebar.Close()
			} else {
				m.aiData.Sidebar.Open(views.ModeAsk)
			}
			m.calcLayout()
			return m, nil
		}

	// AI: agent sidebar (Ctrl+E)
	case "ctrl+e":
		if m.mode == modeNormal && m.fullScreenView == fsNone {
			if m.aiData.Sidebar.Active() {
				m.aiData.Sidebar.Close()
			} else {
				m.aiData.Sidebar.Open(views.ModeAgent)
			}
			m.calcLayout()
			return m, nil
		}

	// Direct panel focus + full-screen view switches
	case "1":
		m.fullScreenView = fsNone
		m.focusedPanel = PanelStatus
		return m, nil
	case "2":
		m.fullScreenView = fsNone
		m.focusedPanel = PanelLog
		return m, nil
	case "3":
		m.fullScreenView = fsPR
		m.ensureGHDataLoaded()
		return m, nil
	case "4":
		m.fullScreenView = fsIssue
		m.ensureGHDataLoaded()
		return m, nil
	case "5":
		m.fullScreenView = fsDashboard
		if !m.dash.Loaded {
			go m.fetchDashboard()
		}
		return m, nil
	case "6":
		m.fullScreenView = fsActions
		m.ensureGHDataLoaded()
		return m, nil
	case "7":
		m.fullScreenView = fsNotes
		m.ensureGHDataLoaded()
		return m, nil
	}

	// If help is open, route keys to help model for search
	if m.showHelp {
		handled := m.helpView.HandleKey(key)
		if !handled {
			// Esc or ? closes help
			m.showHelp = false
			m.helpView.Input = ""
			m.helpView.Cursor = 0
			m.helpView.ShowInput = false
		}
		return m, nil
	}

	// If in full-screen view (PR/Issue), delegate to those handlers
	if m.fullScreenView == fsPR {
		return m.handlePRKeys(key)
	}
	if m.fullScreenView == fsIssue {
		return m.handleIssueKeys(key)
	}
	if m.fullScreenView == fsDashboard {
		return m.handleDashKeys(key)
	}
	if m.fullScreenView == fsActions {
		return m.handleActionsKeys(key)
	}
	if m.fullScreenView == fsNotes {
		return m.handleNotesKeys(key)
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
	case fsPR: // PRs
		if m.prs.PRs == nil {
			go m.fetchPRs()
		}
	case fsIssue: // Issues
		if m.issues.Issues == nil {
			go m.fetchIssues()
		}
	case fsActions: // Actions
		if m.actions.Runs == nil {
			go m.fetchActions()
		}
	case fsNotes: // Notifications
		if !m.notes.Loaded {
			go m.fetchNotes()
		}
	}
}

func (m *Model) refreshGitHubData() {
	if !m.ghDetected || m.gh == nil {
		return
	}
	go m.fetchPRs()
	go m.fetchIssues()
	go m.fetchActions()
	go m.fetchNotes()
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

// fetchDashboard loads recents + favorites with a quick per-repo status.
// Runs async; entries stream in as one update.
func (m *Model) fetchDashboard() {
	cfg, err := config.New()
	if err != nil {
		m.msgs <- teaMsg{view: 5, data: dashEvent{err: err}}
		return
	}
	seen := map[string]bool{}
	var paths []string
	for _, p := range cfg.GetFavoriteRepos() {
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	for _, p := range cfg.GetRecentRepos() {
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	current := ""
	if m.git != nil {
		current = m.git.Path()
	}
	if current != "" && !seen[current] {
		paths = append([]string{current}, paths...)
	}
	favs := map[string]bool{}
	for _, p := range cfg.GetFavoriteRepos() {
		favs[p] = true
	}

	entries := make([]views.DashEntry, 0, len(paths))
	for _, p := range paths {
		e := views.DashEntry{Name: baseName(p), Path: p, Favorite: favs[p], Current: p == current}
		if st, err := quickRepoStatus(p); err != nil {
			e.Err = "unreachable"
		} else {
			e.Branch = st.Branch
			e.Clean = st.IsClean
			e.Dirty = len(st.Files)
			e.Ahead = st.Ahead
			e.Behind = st.Behind
		}
		entries = append(entries, e)
	}
	m.msgs <- teaMsg{view: 5, data: dashEvent{entries: entries}}
}

// quickRepoStatus stats one repo without touching the current exec.
func quickRepoStatus(path string) (*models.Status, error) {
	exec, err := gitpkg.NewNativeExec("")
	if err != nil {
		return nil, err
	}
	if err := exec.Open(path); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()
	return exec.Status(ctx)
}

// switchRepo repoints the TUI at another local repo.
func (m *Model) switchRepo(path string) {
	if m.git == nil {
		m.dash.Error = "no git backend"
		return
	}
	if err := m.git.Open(path); err != nil {
		m.dash.Error = fmt.Sprintf("open %s: %v", path, err)
		return
	}
	if cfg, err := config.New(); err == nil {
		cfg.AddRecentRepo(path)
		_ = cfg.Save()
	}
	m.prs = views.NewPRListModel()
	m.issues = views.NewIssueListModel()
	m.actions = views.NewActionsModel()
	m.notes = views.NewNotesModel()
	m.ghOwner, m.ghRepo, m.ghDetected = "", "", false
	m.tryDetectRepo()
	m.sub.Refresh()
	m.fullScreenView = fsNone
	m.focusedPanel = PanelStatus
	go m.fetchDashboard()
}

// fetchActions loads workflow runs for the detected repo.
func (m *Model) fetchActions() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: 6, data: actionsEvent{err: fmt.Errorf("GitHub not authenticated; run 'zgit auth login'")}}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15e9)
	defer cancel()
	runs, err := m.gh.ListWorkflowRuns(ctx, m.ghOwner, m.ghRepo, github.RunsFilter{Limit: 30})
	if err != nil {
		m.msgs <- teaMsg{view: 6, data: actionsEvent{err: err}}
		return
	}
	if runs == nil {
		runs = []*models.WorkflowRun{}
	}
	m.msgs <- teaMsg{view: 6, data: actionsEvent{runs: runs}}
}

// fetchActionJobs loads jobs for one run (drill-in).
func (m *Model) fetchActionJobs(runID int64) {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: 6, data: actionJobsEvent{runID: runID, err: fmt.Errorf("GitHub not authenticated")}}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15e9)
	defer cancel()
	jobs, err := m.gh.ListWorkflowJobs(ctx, m.ghOwner, m.ghRepo, runID)
	if err != nil {
		m.msgs <- teaMsg{view: 6, data: actionJobsEvent{runID: runID, err: err}}
		return
	}
	if jobs == nil {
		jobs = []*models.Job{}
	}
	m.msgs <- teaMsg{view: 6, data: actionJobsEvent{runID: runID, jobs: jobs}}
}

// fetchNotes loads the global notification inbox.
func (m *Model) fetchNotes() {
	if m.gh == nil {
		m.msgs <- teaMsg{view: 7, data: notesEvent{err: fmt.Errorf("GitHub not authenticated; run 'zgit auth login'")}}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15e9)
	defer cancel()
	items, err := m.gh.ListNotifications(ctx, m.notes.ShowAll, 30)
	if err != nil {
		m.msgs <- teaMsg{view: 7, data: notesEvent{err: err}}
		return
	}
	if items == nil {
		items = []*models.Notification{}
	}
	m.msgs <- teaMsg{view: 7, data: notesEvent{items: items}}
}

// markNoteRead marks one thread read (server + local flip).
func (m *Model) markNoteRead(id string) {
	m.notes.MarkRead(id)
	if m.gh == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10e9)
		defer cancel()
		_ = m.gh.MarkNotificationRead(ctx, id)
	}()
}

// markAllNotesRead marks the whole inbox read.
func (m *Model) markAllNotesRead() {
	m.notes.MarkAllRead()
	if m.gh == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10e9)
		defer cancel()
		_ = m.gh.MarkAllNotificationsRead(ctx)
	}()
}

// baseName returns the last path segment.
func baseName(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// --- View-specific key handlers ---

func (m *Model) handleStatusKeys(key string) (tea.Model, tea.Cmd) {
	visible := m.status.VisibleFiles()
	totalVisible := len(visible)

	switch key {
	case "j", "down":
		if totalVisible > 0 && m.status.Cursor < totalVisible-1 {
			m.status.Cursor++
		}
	case "k", "up":
		if m.status.Cursor > 0 {
			m.status.Cursor--
		}
	case "g", "home":
		m.status.Cursor = 0
	case "G", "end":
		m.status.Cursor = totalVisible - 1
		if m.status.Cursor < 0 {
			m.status.Cursor = 0
		}
	case "enter":
		if m.status.CursorFile() != nil {
			m.openFileDiff()
		}
	case " ":
		m.stageToggle()
	case "s":
		m.stageFile()
	case "S":
		m.unstageFile()
	case "a":
		m.stageAll()
	case "A":
		m.unstageAll()
	case "d":
		m.discardFile()
	case "c":
		m.openCommitDialog()
	case "P":
		m.pushAfterCommit = true
		m.openCommitDialog()
	}
	return m, nil
}

// --- File operations ---

func (m *Model) stageToggle() {
	f := m.status.CursorFile()
	if f == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()

	if f.Staged != models.StatusUnmodified && f.Staged != models.StatusUntracked && f.Unstaged == models.StatusUnmodified {
		// Only staged → unstage
		if err := m.git.RestoreStaged(ctx, f.Path); err != nil {
			m.status.Error = fmt.Sprintf("unstage %s: %v", f.Path, err)
			return
		}
		m.status.Status.OptimisticUnstage(f.Path)
	} else {
		// Has unstaged or untracked → stage
		if err := m.git.Add(ctx, gitpkg.AddOptions{}, f.Path); err != nil {
			m.status.Error = fmt.Sprintf("stage %s: %v", f.Path, err)
			return
		}
		m.status.Status.OptimisticStage(f.Path)
	}
	m.sub.Refresh()
}

func (m *Model) stageFile() {
	f := m.status.CursorFile()
	if f == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()
	if err := m.git.Add(ctx, gitpkg.AddOptions{}, f.Path); err != nil {
		m.status.Error = fmt.Sprintf("stage %s: %v", f.Path, err)
		return
	}
	m.status.Status.OptimisticStage(f.Path)
	m.sub.Refresh()
}

func (m *Model) unstageFile() {
	f := m.status.CursorFile()
	if f == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()
	if err := m.git.RestoreStaged(ctx, f.Path); err != nil {
		m.status.Error = fmt.Sprintf("unstage %s: %v", f.Path, err)
		return
	}
	m.status.Status.OptimisticUnstage(f.Path)
	m.sub.Refresh()
}

func (m *Model) discardFile() {
	f := m.status.CursorFile()
	if f == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()
	if err := m.git.Restore(ctx, f.Path); err != nil {
		m.status.Error = fmt.Sprintf("discard %s: %v", f.Path, err)
		return
	}
	// Remove from local list optimistically
	for i := range m.status.Status.Files {
		if m.status.Status.Files[i].Path == f.Path {
			m.status.Status.Files = append(m.status.Status.Files[:i], m.status.Status.Files[i+1:]...)
			break
		}
	}
	m.status.Status.Recount()
	if m.status.Cursor >= len(m.status.Status.Files) {
		m.status.Cursor = len(m.status.Status.Files) - 1
		if m.status.Cursor < 0 {
			m.status.Cursor = 0
		}
	}
	m.sub.Refresh()
}

func (m *Model) stageAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 30e9)
	defer cancel()
	if err := m.git.Add(ctx, gitpkg.AddOptions{All: true}); err != nil {
		m.status.Error = fmt.Sprintf("stage all: %v", err)
		return
	}
	m.status.Status.OptimisticStageAll()
	m.sub.Refresh()
}

func (m *Model) unstageAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 30e9)
	defer cancel()
	if err := m.git.Reset(ctx); err != nil {
		m.status.Error = fmt.Sprintf("unstage all: %v", err)
		return
	}
	m.status.Status.OptimisticUnstageAll()
	m.sub.Refresh()
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
	case "enter":
		if m.log.Cursor >= 0 && m.log.Cursor < len(m.log.Commits) {
			m.openCommitDiff(m.log.Commits[m.log.Cursor])
		}
	case "C":
		m.cherryPickCommit()
	case "r":
		m.status.Error = "interactive rebase not yet implemented in TUI (use CLI: zgit rebase)"
	}
	return m, nil
}

func (m *Model) cherryPickCommit() {
	if m.log.Cursor < 0 || m.log.Cursor >= len(m.log.Commits) {
		return
	}
	c := m.log.Commits[m.log.Cursor]
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()

	if err := m.git.CherryPickNoCommit(ctx, c.Hash); err != nil {
		m.log.Error = fmt.Sprintf("cherry-pick %s: %v", c.Hash[:7], err)
		return
	}
	m.sub.Refresh()
}

func (m *Model) openCommitDiff(commit *models.Commit) {
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()

	// Show diff of this commit against its parent
	parent := commit.Hash + "^"
	if len(commit.Parents) > 0 {
		parent = commit.Parents[0]
	}

	opts := gitpkg.DiffOptions{
		A:       parent,
		B:       commit.Hash,
		Unified: true,
	}
	diff, err := m.git.Diff(ctx, opts)
	if err != nil {
		m.log.Error = fmt.Sprintf("show %s: %v", commit.Hash[:7], err)
		return
	}

	// Build combined diff text from files
	var fullDiff string
	totalAdds, totalDels := 0, 0
	for _, f := range diff.Files {
		totalAdds += f.Additions
		totalDels += f.Deletions
		fullDiff += f.UnifiedDiff + "\n"
	}

	m.diffViewer.SetDiff(commit.Hash[:7]+": "+commit.Message, fullDiff, totalAdds, totalDels)
	m.mode = modeDiff
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
	// Filter input mode captures all typing.
	if m.prs.Filtering {
		switch key {
		case "esc", "enter":
			m.prs.Filtering = false
			if key == "esc" {
				m.prs.Filter = ""
			}
			m.prs.ClampCursor()
			return m, nil
		case "backspace":
			if len(m.prs.Filter) > 0 {
				m.prs.Filter = m.prs.Filter[:len(m.prs.Filter)-1]
				m.prs.ClampCursor()
			}
			return m, nil
		default:
			if len(key) == 1 || key == " " {
				m.prs.Filter += key
				m.prs.ClampCursor()
			}
			return m, nil
		}
	}
	switch key {
	case "/":
		m.prs.Filtering = true
		return m, nil
	case "j", "down":
		if len(m.prs.Filtered()) > 0 && m.prs.Cursor < len(m.prs.Filtered())-1 {
			m.prs.Cursor++
		}
	case "k", "up":
		if m.prs.Cursor > 0 {
			m.prs.Cursor--
		}
	case "g", "home":
		m.prs.Cursor = 0
	case "G", "end":
		m.prs.Cursor = len(m.prs.Filtered()) - 1
		if m.prs.Cursor < 0 {
			m.prs.Cursor = 0
		}
	case "pgdown", "ctrl+f":
		m.prs.Cursor += m.prs.Height
		if m.prs.Cursor >= len(m.prs.Filtered()) {
			m.prs.Cursor = len(m.prs.Filtered()) - 1
		}
	case "pgup", "ctrl+b":
		m.prs.Cursor -= m.prs.Height
		if m.prs.Cursor < 0 {
			m.prs.Cursor = 0
		}
	case "c", "o":
		// Create PR — get current branch
		branch := m.currentBranch()
		if branch == "" {
			m.prs.Error = "cannot determine current branch"
			return m, nil
		}
		m.prCreateDlg.Open(branch, "main")
		m.mode = modePRCreate
	case "m":
		// Merge selected PR
		pr := m.prs.SelectedPR()
		if pr == nil {
			m.prs.Error = "no PR selected"
			return m, nil
		}
		if pr.State != models.PRStateOpen {
			m.prs.Error = "only open PRs can be merged"
			return m, nil
		}
		m.prMergeDlg.Open(pr)
		m.mode = modePRMerge
	case "r":
		go m.fetchPRs()
	}
	return m, nil
}

// currentBranch returns the current git branch name.
func (m *Model) currentBranch() string {
	if m.status.Status != nil && m.status.Status.Branch != "" {
		return m.status.Status.Branch
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5e9)
	defer cancel()
	branch, err := m.git.CurrentBranch(ctx)
	if err != nil {
		return ""
	}
	return branch
}

func (m *Model) handleIssueKeys(key string) (tea.Model, tea.Cmd) {
	// Filter input mode captures all typing.
	if m.issues.Filtering {
		switch key {
		case "esc", "enter":
			m.issues.Filtering = false
			if key == "esc" {
				m.issues.Filter = ""
			}
			m.issues.ClampCursor()
			return m, nil
		case "backspace":
			if len(m.issues.Filter) > 0 {
				m.issues.Filter = m.issues.Filter[:len(m.issues.Filter)-1]
				m.issues.ClampCursor()
			}
			return m, nil
		default:
			if len(key) == 1 || key == " " {
				m.issues.Filter += key
				m.issues.ClampCursor()
			}
			return m, nil
		}
	}
	switch key {
	case "/":
		m.issues.Filtering = true
		return m, nil
	case "j", "down":
		if len(m.issues.Filtered()) > 0 && m.issues.Cursor < len(m.issues.Filtered())-1 {
			m.issues.Cursor++
		}
	case "k", "up":
		if m.issues.Cursor > 0 {
			m.issues.Cursor--
		}
	case "g", "home":
		m.issues.Cursor = 0
	case "G", "end":
		m.issues.Cursor = len(m.issues.Filtered()) - 1
		if m.issues.Cursor < 0 {
			m.issues.Cursor = 0
		}
	case "pgdown", "ctrl+f":
		m.issues.Cursor += m.issues.Height
		if m.issues.Cursor >= len(m.issues.Filtered()) {
			m.issues.Cursor = len(m.issues.Filtered()) - 1
		}
	case "pgup", "ctrl+b":
		m.issues.Cursor -= m.issues.Height
		if m.issues.Cursor < 0 {
			m.issues.Cursor = 0
		}
	}
	return m, nil
}

// filterKeys routes typing to a filterable list model. Returns true when
// the key was consumed by filter input.
func filterKeys(key string, filtering *bool, filter *string, clamp func()) bool {
	if *filtering {
		switch key {
		case "esc", "enter":
			*filtering = false
			if key == "esc" {
				*filter = ""
			}
			clamp()
			return true
		case "backspace":
			if len(*filter) > 0 {
				*filter = (*filter)[:len(*filter)-1]
				clamp()
			}
			return true
		default:
			if len(key) == 1 || key == " " {
				*filter += key
				clamp()
			}
			return true
		}
	}
	return false
}

func (m *Model) handleDashKeys(key string) (tea.Model, tea.Cmd) {
	if filterKeys(key, &m.dash.Filtering, &m.dash.Filter, m.dash.ClampCursor) {
		return m, nil
	}
	switch key {
	case "/":
		m.dash.Filtering = true
		return m, nil
	case "j", "down":
		if len(m.dash.Filtered()) > 0 && m.dash.Cursor < len(m.dash.Filtered())-1 {
			m.dash.Cursor++
		}
	case "k", "up":
		if m.dash.Cursor > 0 {
			m.dash.Cursor--
		}
	case "g", "home":
		m.dash.Cursor = 0
	case "G", "end":
		m.dash.Cursor = len(m.dash.Filtered()) - 1
		if m.dash.Cursor < 0 {
			m.dash.Cursor = 0
		}
	case "enter":
		if sel := m.dash.Selected(); sel != nil {
			m.switchRepo(sel.Path)
		}
	case "r":
		m.dash.Loaded = false
		go m.fetchDashboard()
	}
	return m, nil
}

func (m *Model) handleActionsKeys(key string) (tea.Model, tea.Cmd) {
	// In job drill-in, backspace/esc returns to runs.
	if m.actions.JobsFor != 0 && (key == "backspace" || key == "esc") {
		m.actions.Jobs = nil
		m.actions.JobsFor = 0
		m.actions.Cursor = 0
		m.actions.Offset = 0
		return m, nil
	}
	// Filter input captures all typing (runs list only).
	if m.actions.JobsFor == 0 && m.actions.Filtering {
		filterKeys(key, &m.actions.Filtering, &m.actions.Filter, m.actions.ClampCursor)
		return m, nil
	}
	n := len(m.actions.Filtered())
	if m.actions.JobsFor != 0 {
		n = len(m.actions.Jobs)
	}
	switch key {
	case "/":
		if m.actions.JobsFor == 0 {
			m.actions.Filtering = true
		}
		return m, nil
	case "j", "down":
		if n > 0 && m.actions.Cursor < n-1 {
			m.actions.Cursor++
		}
	case "k", "up":
		if m.actions.Cursor > 0 {
			m.actions.Cursor--
		}
	case "g", "home":
		m.actions.Cursor = 0
	case "G", "end":
		m.actions.Cursor = n - 1
		if m.actions.Cursor < 0 {
			m.actions.Cursor = 0
		}
	case "enter":
		if m.actions.JobsFor == 0 {
			if run := m.actions.SelectedRun(); run != nil {
				go m.fetchActionJobs(run.ID)
			}
		}
	case "r":
		m.actions.Runs = nil
		go m.fetchActions()
	}
	return m, nil
}

func (m *Model) handleNotesKeys(key string) (tea.Model, tea.Cmd) {
	if filterKeys(key, &m.notes.Filtering, &m.notes.Filter, m.notes.ClampCursor) {
		return m, nil
	}
	switch key {
	case "/":
		m.notes.Filtering = true
		return m, nil
	case "j", "down":
		if len(m.notes.Ordered()) > 0 && m.notes.Cursor < len(m.notes.Ordered())-1 {
			m.notes.Cursor++
		}
	case "k", "up":
		if m.notes.Cursor > 0 {
			m.notes.Cursor--
		}
	case "g", "home":
		m.notes.Cursor = 0
	case "G", "end":
		m.notes.Cursor = len(m.notes.Ordered()) - 1
		if m.notes.Cursor < 0 {
			m.notes.Cursor = 0
		}
	case "x":
		if sel := m.notes.Selected(); sel != nil && sel.Unread {
			m.markNoteRead(sel.ID)
		}
	case "X":
		m.markAllNotesRead()
	case "u":
		// Toggle unread/all (server-side) and reload.
		m.notes.ShowAll = !m.notes.ShowAll
		m.notes.Loaded = false
		go m.fetchNotes()
	case "r":
		m.notes.Loaded = false
		go m.fetchNotes()
	}
	return m, nil
}

// --- Diff & Commit ---

func (m *Model) openFileDiff() {
	file := m.status.CursorFile()
	if file == nil {
		return
	}

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
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			if m.commitDlg.State == views.CommitConfirming {
				m.commitDlg.State = views.CommitResult
				m.commitDlg.Result = views.CommitResultInfo{
					Success: true,
					Message: m.commitDlg.GetMessage(),
				}
				doPush := m.pushAfterCommit
				m.pushAfterCommit = false
				go m.executeCommit(m.commitDlg.GetMessage(), doPush)
				return m, nil
			}
		case "ctrl+g":
			// AI generate commit message from staged diff
			if m.commitDlg.State == views.CommitInputSubject || m.commitDlg.State == views.CommitInputBody {
				go m.generateAICommitMsg()
				return m, nil
			}
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

func (m *Model) executeCommit(msg string, doPush bool) {
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

	// Push after commit if requested
	pushErrMsg := ""
	if doPush {
		branch := "HEAD"
		if commits, logErr := m.git.Log(ctx, gitpkg.LogOptions{Count: 1}); logErr == nil && len(commits) > 0 {
			// Keep the hash we already got
		}
		if m.status.Status != nil && m.status.Status.Branch != "" {
			branch = m.status.Status.Branch
		}
		pushOpts := gitpkg.PushOptions{
			SetUpstream: true,
			Branch:      branch,
		}
		if pErr := m.git.Push(ctx, pushOpts); pErr != nil {
			pushErrMsg = fmt.Sprintf("commit OK, push failed: %v", pErr)
		}
	}

	m.msgs <- teaMsg{
		view: -1,
		data: commitResultEvent{
			Result: views.CommitResultInfo{
				Success: true,
				Hash:    hash,
				Message: msg,
				Error:   pushErrMsg,
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

// --- PR Create Dialog ---

func (m *Model) updatePRCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		if m.prCreateDlg.State == views.PRCreateConfirming {
			m.prCreateDlg.State = views.PRCreateResult
			m.prCreateDlg.Result = views.PRCreateResultInfo{Success: true}
			go m.executeCreatePR()
			return m, nil
		}
	}

	updated, cmd := m.prCreateDlg.Update(msg)
	m.prCreateDlg = *updated

	if !m.prCreateDlg.Active() {
		m.mode = modeNormal
		return m, nil
	}

	if m.prCreateDlg.State == views.PRCreateResult {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter", "esc":
				m.prCreateDlg.Close()
				m.mode = modeNormal
				go m.fetchPRs() // refresh list
				return m, nil
			}
		}
	}

	return m, cmd
}

func (m *Model) executeCreatePR() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: -4, data: prCreateResultEvent{
			Result: views.PRCreateResultInfo{
				Success: false,
				Error:   "GitHub not authenticated",
			},
		}}
		return
	}

	req := m.prCreateDlg.GetRequest()
	ctx, cancel := context.WithTimeout(context.Background(), 30e9)
	defer cancel()

	pr, err := m.gh.CreatePullRequest(ctx, m.ghOwner, m.ghRepo, req)
	if err != nil {
		m.msgs <- teaMsg{view: -4, data: prCreateResultEvent{
			Result: views.PRCreateResultInfo{
				Success: false,
				Error:   err.Error(),
			},
		}}
		return
	}

	m.msgs <- teaMsg{view: -4, data: prCreateResultEvent{
		Result: views.PRCreateResultInfo{
			Success: true,
			Number:  pr.Number,
			URL:     fmt.Sprintf("https://github.com/%s/%s/pull/%d", m.ghOwner, m.ghRepo, pr.Number),
		},
	}}
}

// --- PR Merge Dialog ---

func (m *Model) updatePRMerge(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		if m.prMergeDlg.State == views.PRMergeConfirming {
			m.prMergeDlg.State = views.PRMergeResult
			m.prMergeDlg.Result = views.PRMergeResultInfo{Success: true}
			go m.executeMergePR()
			return m, nil
		}
	}

	updated, cmd := m.prMergeDlg.Update(msg)
	m.prMergeDlg = *updated

	if !m.prMergeDlg.Active() {
		m.mode = modeNormal
		return m, nil
	}

	if m.prMergeDlg.State == views.PRMergeResult {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter", "esc":
				m.prMergeDlg.Close()
				m.mode = modeNormal
				go m.fetchPRs() // refresh list
				return m, nil
			}
		}
	}

	return m, cmd
}

func (m *Model) executeMergePR() {
	if m.gh == nil || !m.ghDetected {
		m.msgs <- teaMsg{view: -5, data: prMergeResultEvent{
			Result: views.PRMergeResultInfo{
				Success: false,
				Error:   "GitHub not authenticated",
			},
		}}
		return
	}

	pr := m.prMergeDlg.PR
	if pr == nil {
		m.msgs <- teaMsg{view: -5, data: prMergeResultEvent{
			Result: views.PRMergeResultInfo{
				Success: false,
				Error:   "no PR selected",
			},
		}}
		return
	}

	method := m.prMergeDlg.Strategy.APIValue()
	ctx, cancel := context.WithTimeout(context.Background(), 30e9)
	defer cancel()

	err := m.gh.MergePullRequest(ctx, m.ghOwner, m.ghRepo, pr.Number, method)
	if err != nil {
		m.msgs <- teaMsg{view: -5, data: prMergeResultEvent{
			Result: views.PRMergeResultInfo{
				Success: false,
				Error:   err.Error(),
			},
		}}
		return
	}

	m.msgs <- teaMsg{view: -5, data: prMergeResultEvent{
		Result: views.PRMergeResultInfo{
			Success: true,
			Message: fmt.Sprintf("PR #%d merged via %s", pr.Number, m.prMergeDlg.Strategy.String()),
		},
	}}
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

type dashEvent struct {
	entries []views.DashEntry
	err     error
}

type actionsEvent struct {
	runs []*models.WorkflowRun
	err  error
}

type actionJobsEvent struct {
	runID int64
	jobs  []*models.Job
	err   error
}

type notesEvent struct {
	items []*models.Notification
	err   error
}

type prCreateResultEvent struct {
	Result views.PRCreateResultInfo
}

type prMergeResultEvent struct {
	Result views.PRMergeResultInfo
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

	// AI commit message generation result
	if msg.view == aiCommitMsgViewID {
		if evt, ok := msg.data.(aiCommitMsgResult); ok {
			if evt.Success {
				m.populateCommitFromAI(evt.Message)
			} else {
				m.commitDlg.Result = views.CommitResultInfo{
					Success: false,
					Error:   evt.Error,
				}
				m.commitDlg.State = views.CommitResult
			}
		}
		return m, nil
	}

	if msg.view == -4 {
		if evt, ok := msg.data.(prCreateResultEvent); ok {
			m.prCreateDlg.Result = evt.Result
			m.prCreateDlg.State = views.PRCreateResult
		}
		return m, nil
	}

	if msg.view == -5 {
		if evt, ok := msg.data.(prMergeResultEvent); ok {
			m.prMergeDlg.Result = evt.Result
			m.prMergeDlg.State = views.PRMergeResult
		}
		return m, nil
	}

	// AI streaming messages
	if msg.view == aiStreamViewID {
		m.handleAIStreamMsg(msg.data)
		return m, nil
	}

	// Config validation result
	if msg.view == configViewID {
		if res, ok := msg.data.(configResultMsg); ok {
			m.configDlg.SetResult(res.success, res.msg)
			if res.success {
				// Update ghUser from config
				cfg, err := config.New()
				if err == nil {
					m.ghUser = cfg.GetString("github.user")
				}
				// Re-init GitHub client with new token
				m.tryInitGitHub()
			}
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
	case 5: // Dashboard
		if evt, ok := msg.data.(dashEvent); ok {
			if evt.err != nil {
				m.dash.Error = evt.err.Error()
				m.dash.Loaded = true
			} else if evt.entries != nil {
				m.dash.UpdateEntries(evt.entries)
			}
		}
	case 6: // Actions
		if evt, ok := msg.data.(actionsEvent); ok {
			if evt.err != nil {
				m.actions.Error = evt.err.Error()
			} else if evt.runs != nil {
				m.actions.UpdateRuns(evt.runs)
			}
		} else if evt, ok := msg.data.(actionJobsEvent); ok {
			if evt.err != nil {
				m.actions.Error = evt.err.Error()
			} else if evt.jobs != nil {
				m.actions.UpdateJobs(evt.runID, evt.jobs)
			}
		}
	case 7: // Notifications
		if evt, ok := msg.data.(notesEvent); ok {
			if evt.err != nil {
				m.notes.Error = evt.err.Error()
				m.notes.Loaded = true
			} else if evt.items != nil {
				m.notes.UpdateNotes(evt.items)
			}
		}
	}
	return m, nil
}

// overlayCenter places a modal string dead-center over a base view string.
// Strips ANSI from base lines before slicing to prevent broken escape sequences.
// termWidth and termHeight define the full terminal dimensions used for centering.
func overlayCenter(base, modal string, termWidth, termHeight int) string {
	if modal == "" {
		return base
	}

	baseLines := strings.Split(base, "\n")
	modalLines := strings.Split(modal, "\n")

	if len(modalLines) == 0 {
		return base
	}

	// Calculate modal visual dimensions
	modalH := len(modalLines)
	modalW := 0
	for _, l := range modalLines {
		w := lipgloss.Width(l)
		if w > modalW {
			modalW = w
		}
	}

	if modalW <= 0 || modalH <= 0 {
		return base
	}

	// Center coordinates
	startY := (termHeight - modalH) / 2
	if startY < 0 {
		startY = 0
	}
	startX := (termWidth - modalW) / 2
	if startX < 0 {
		startX = 0
	}

	result := make([]string, len(baseLines))
	for i, line := range baseLines {
		if i >= startY && i < startY+modalH && i < len(baseLines) {
			// Strip ANSI from base line so slicing never cuts escape codes
			plain := stripANSI(line)
			plainRunes := []rune(plain)

			// Pad runes to termWidth so left/right slicing is safe (rune index = visual column for ASCII/narrow chars)
			if len(plainRunes) < termWidth {
				padded := make([]rune, termWidth)
				for i := 0; i < termWidth; i++ {
					if i < len(plainRunes) {
						padded[i] = plainRunes[i]
					} else {
						padded[i] = ' '
					}
				}
				plainRunes = padded
			}

			modalLine := modalLines[i-startY]
			modalWidth := lipgloss.Width(modalLine)

			// Left part (base content before modal) — rune-indexed to avoid multibyte slicing errors
			left := string(plainRunes[:startX])

			// Right part (base content after modal)
			rightEnd := startX + modalWidth
			if rightEnd < len(plainRunes) {
				right := string(plainRunes[rightEnd:])
				result[i] = left + modalLine + right
			} else {
				result[i] = left + modalLine
			}

			// Ensure result fills termWidth (modal may be narrower)
			if w := lipgloss.Width(result[i]); w < termWidth {
				result[i] += strings.Repeat(" ", termWidth-w)
			}
		} else {
			result[i] = line // keep original styled line
		}
	}

	return strings.Join(result, "\n")
}

// stripANSI removes ANSI escape sequences from a string.
// ANSI sequences start with \x1b (ESC) followed by '[', parameters, and a letter (a-z, A-Z).
func stripANSI(s string) string {
	var out strings.Builder
	inEscape := false
	for _, r := range s {
		if inEscape {
			// ANSI escape sequence ends on any ASCII letter (a-z, A-Z), tilde, or @
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' || r == '@' {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
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
