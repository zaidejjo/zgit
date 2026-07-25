// Package tui provides the Bubble Tea-based terminal UI for zgit.
package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// View IDs for tab switching.
const (
	ViewStatus = 0
	ViewLog    = 1
	ViewBranch = 2
	viewCount  = 3 // total number of views (excludes help)
)

// tab labels for the header bar.
var tabNames = []string{"Status", "Log", "Branches"}

// Model is the root Bubble Tea model for the zgit TUI.
type Model struct {
	// Dependencies
	git  *git.NativeExec
	sub  *Subscriber
	msgs chan teaMsg

	// State
	activeView int  // currently visible view (0=status, 1=log, 2=branches)
	showHelp   bool // help overlay visible
	ready      bool // terminal dimensions known
	quitting   bool

	// Sub-models
	status   views.StatusModel
	log      views.LogModel
	branches views.BranchModel
	helpView views.HelpModel

	// Bubble Tea components
	help     help.Model
	viewport viewport.Model

	// Terminal dimensions
	width  int
	height int
}

// NewModel creates the root TUI model.
func NewModel(gitExec *git.NativeExec) *Model {
	msgs := make(chan teaMsg, 16)

	return &Model{
		git:      gitExec,
		msgs:     msgs,
		sub:      NewSubscriber(gitExec, msgs),
		status:   views.NewStatusModel(),
		log:      views.NewLogModel(),
		branches: views.NewBranchModel(),
		helpView: views.HelpModel{},
		help:     help.New(),
	}
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

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case teaMsg:
		return m.handleEngineMsg(msg)

	default:
		return m, nil
	}
}

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return styles.LoadingStyle.Render("Loading...")
	}

	if m.quitting {
		return ""
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
	return styles.AppStyle.Width(width).Render("") // placeholder
}

// renderContent returns the content area for the active view.
func (m *Model) renderContent() string {
	if m.showHelp {
		return m.helpView.View(m.width)
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
	right := styles.StatusBarInfoStyle.Render(" ? help • q quit ")

	bar := styles.StatusBarStyle.
		Width(m.width).
		Render(left + right)

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
		return m, nil

	case "tab":
		m.activeView = (m.activeView + 1) % viewCount
		return m, nil

	case "shift+tab":
		m.activeView = (m.activeView - 1 + viewCount) % viewCount
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
	}

	return m, nil
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
	}
	return m, nil
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
			if err := m.git.Checkout(nil, b.Name); err != nil {
				m.branches.Error = fmt.Sprintf("checkout %s: %v", b.Name, err)
			} else {
				m.sub.Refresh()
			}
		}
	}
	return m, nil
}

// handleEngineMsg processes messages from the background subscriber.
func (m *Model) handleEngineMsg(msg teaMsg) (tea.Model, tea.Cmd) {
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
func Run(gitExec *git.NativeExec) error {
	m := NewModel(gitExec)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		return err
	}
	return nil
}
