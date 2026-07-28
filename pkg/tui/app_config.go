package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaidejjo/zgit/pkg/core/config"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// updateConfig handles the config dialog lifecycle.
func (m *Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		handled := m.configDlg.HandleKey(key)
		if !handled {
			return m, nil
		}

		// On enter in input state, save and validate
		if key == "enter" && m.configDlg.State == views.ConfigLoading {
			m.saveConfigValue()
			return m, nil
		}

		// If dialog was closed, exit mode
		if !m.configDlg.Active() {
			m.mode = modeNormal
		}
	}

	return m, nil
}

// saveConfigValue saves the entered value and validates if applicable.
func (m *Model) saveConfigValue() {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	value := m.configDlg.GetInput()

	switch m.configDlg.Field {
	case views.FieldGitHubToken:
		cfg.Set("github.token", value)
		if err := cfg.Save(); err != nil {
			m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
			return
		}
		// Validate token by fetching user
		go m.validateGitHubToken(cfg, value)

	case views.FieldAIProvider:
		cfg.Set("ai.provider", value)
		if err := cfg.Save(); err != nil {
			m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
			return
		}
		m.aiData.Provider = value
		m.configDlg.SetResult(true, "Provider saved")

	case views.FieldAIModel:
		cfg.Set("ai.model", value)
		if err := cfg.Save(); err != nil {
			m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
			return
		}
		m.aiData.Model = value
		m.aiData.Sidebar.Model = value
		m.configDlg.SetResult(true, "Model saved")

	case views.FieldAIAPIKey:
		cfg.Set("ai.api_key", value)
		if err := cfg.Save(); err != nil {
			m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
			return
		}
		m.aiData.APIKey = value
		m.configDlg.SetResult(true, "API key saved")
	}
}

// validateGitHubToken tries to fetch the authenticated user to verify the token.
func (m *Model) validateGitHubToken(cfg *config.Manager, token string) {
	gh, err := github.NewCombinedClient(token)
	if err != nil {
		m.msgs <- teaMsg{view: configViewID, data: configResultMsg{success: false, msg: fmt.Sprintf("Client error: %v", err)}}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()

	user, err := gh.GetAuthenticatedUser(ctx)
	if err != nil {
		m.msgs <- teaMsg{view: configViewID, data: configResultMsg{success: false, msg: fmt.Sprintf("Validation failed: %v", err)}}
		return
	}

	// Save user to config
	cfg.Set("github.user", user.Login)
	if saveErr := cfg.Save(); saveErr != nil {
		m.msgs <- teaMsg{view: configViewID, data: configResultMsg{success: false, msg: fmt.Sprintf("Token valid but save failed: %v", saveErr)}}
		return
	}

	m.msgs <- teaMsg{view: configViewID, data: configResultMsg{
		success: true,
		msg:     fmt.Sprintf("Authenticated as @%s", user.Login),
	}}
}

// configResultMsg carries the outcome of a config operation.
type configResultMsg struct {
	success bool
	msg     string
}

const configViewID = -3
