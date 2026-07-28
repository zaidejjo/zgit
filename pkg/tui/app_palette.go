package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// updatePalette handles the command palette lifecycle.
func (m *Model) updatePalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		handled := m.palette.HandleKey(key)
		if !handled {
			return m, nil
		}

		// On enter, execute selected command
		if key == "enter" {
			cmd := m.palette.SelectedCommand()
			if cmd != nil {
				m.palette.Close()
				m.mode = modeNormal
				return m.executePaletteCommand(cmd.ID)
			}
			return m, nil
		}

		// If palette was closed
		if !m.palette.Active() {
			m.mode = modeNormal
		}
	}

	return m, nil
}

// executePaletteCommand translates a command ID into an action and returns a tea.Cmd.
func (m *Model) executePaletteCommand(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "stage-all":
		m.stageAll()
	case "unstage-all":
		m.unstageAll()
	case "commit":
		m.openCommitDialog()
	case "commit-push":
		m.pushAfterCommit = true
		m.openCommitDialog()
	case "cherry-pick":
		// Switch to log panel first
		m.focusedPanel = PanelLog
		if m.log.Cursor >= 0 && m.log.Cursor < len(m.log.Commits) {
			m.cherryPickCommit()
		}
	case "new-branch":
		m.focusedPanel = PanelBranch
	case "merge-branch":
		m.focusedPanel = PanelBranch
	case "delete-branch":
		m.focusedPanel = PanelBranch
	case "ai-ask":
		m.aiData.Sidebar.Open(views.ModeAsk)
		m.aiData.Sidebar.Width = 44
		m.aiData.Sidebar.Height = m.contentHeight
	case "ai-agent":
		m.aiData.Sidebar.Open(views.ModeAgent)
		m.aiData.Sidebar.Width = 44
		m.aiData.Sidebar.Height = m.contentHeight
	case "open-config":
		m.configDlg.OpenGitHubToken()
		m.mode = modeConfig
	case "next-panel":
		m.focusedPanel = (m.focusedPanel + 1) % panelCount
	case "prev-panel":
		m.focusedPanel = (m.focusedPanel - 1 + panelCount) % panelCount
	case "help":
		m.showHelp = true
		m.helpView.Input = ""
		m.helpView.Cursor = 0
		m.helpView.ShowInput = false
	case "refresh":
		m.sub.Refresh()
		m.refreshGitHubData()
	case "quit":
		m.quitting = true
		m.sub.Stop()
		return m, tea.Quit
	}
	return m, nil
}
