package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaidejjo/zgit/pkg/core/config"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/tui/views"
)

// configViewID is the subscriber view ID for config results.
const configViewID = -3

// configResultMsg carries the outcome of a config operation.
type configResultMsg struct {
	success bool
	msg     string
}

// updateConfig handles the config dialog lifecycle.
func (m *Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Detect bracketed paste (Ctrl+Shift+V) — handle BEFORE normal key processing
		// to avoid double-inserting characters.
		if msg.Paste && m.configDlg.State == views.ConfigInput {
			m.configDlg.HandlePaste(string(msg.Runes))
			return m, nil
		}

		key := msg.String()
		handled := m.configDlg.HandleKey(key)
		if !handled {
			return m, nil
		}

		// On enter in loading state, process selection/value
		if key == "enter" && m.configDlg.State == views.ConfigLoading {
			m.processConfigAction()
			return m, nil
		}

		// On enter in result state, advance or close
		if key == "enter" && m.configDlg.State == views.ConfigResult && m.configDlg.Success {
			// If "saved" result during provider config, advance to next step
			if m.configDlg.Result == "saved" && m.configDlg.TargetProvider != "" {
				m.advanceProviderConfig()
				return m, nil
			}
		}

		// If dialog was closed, exit mode
		if !m.configDlg.Active() {
			m.mode = modeNormal
		}
	}

	return m, nil
}

// processConfigAction handles Enter at the saving step.
func (m *Model) processConfigAction() {
	switch m.configDlg.Field {
	case views.FieldGitHubToken:
		m.saveGitHubToken()

	case views.FieldAIProviderList:
		// User selected a provider to configure or "Add new"
		selected := m.configDlg.GetSelectedProvider()
		if strings.HasPrefix(m.configDlg.GetSelectedItem(), "＋") {
			// Open provider picker
			m.configDlg.OpenAIProviderPicker()
		} else {
			// Configure existing provider
			m.openProviderConfigFlow(selected)
		}

	case views.FieldAIProviderMenu:
		item := m.configDlg.GetSelectedItem()
		// Skip separator line
		if item == "───" {
			m.configDlg.State = views.ConfigSelect // stay in select
			return
		}
		if item == "Add custom provider..." {
			// Show full provider list
			m.configDlg.OpenAIProviderFullList()
			return
		}
		m.openProviderConfigFlow(item)

	case views.FieldAIProviderActions:
		item := m.configDlg.GetSelectedItem()
		if item == "───" || item == "Done" {
			m.refreshProviderList()
			return
		}
		switch item {
		case "Edit API Key":
			m.configDlg.OpenAIProviderKey(m.configDlg.TargetProvider)
		case "Change Model":
			provider := m.configDlg.TargetProvider
			cfg, _ := config.New()
			customModels := cfg.GetCustomModels(provider)
			m.configDlg.OpenAIProviderModel(provider, views.ProviderModelOptions[provider], customModels)
		case "Change Endpoint":
			provider := m.configDlg.TargetProvider
			pc := config.ProviderConfig{}
			for _, kp := range views.KnownProviders {
				if kp.Name == provider {
					pc.Endpoint = kp.DefaultEndpoint
					break
				}
			}
			m.configDlg.OpenAIProviderEndpoint(provider, pc.Endpoint)
		}

	case views.FieldAIAPIKey:
		m.saveProviderAPIKey()

	case views.FieldAIModel:
		item := m.configDlg.GetSelectedItem()
		if views.IsCustomModelOption(item) {
			// Open free-form text input for custom model
			m.configDlg.State = views.ConfigInput
			m.configDlg.Field = views.FieldAIModel
			m.configDlg.Input = ""
			m.configDlg.Cursor = 0
			m.configDlg.Result = ""
			m.configDlg.Success = false
			return
		}
		m.saveProviderModel()

	case views.FieldAIEndpoint:
		m.saveProviderEndpoint()
	}
}

// openProviderConfigFlow starts the multi-step provider configuration.
func (m *Model) openProviderConfigFlow(provider string) {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	pc := cfg.GetProviderConfig(provider)

	// If provider already has a key, show action menu
	if pc.APIKey != "" {
		m.configDlg.OpenAIProviderActions(provider)
		return
	}

	// Step 1: Enter API key
	m.configDlg.OpenAIProviderKey(provider)
}

// advanceProviderConfig moves to the next step in provider config.
func (m *Model) advanceProviderConfig() {
	provider := m.configDlg.TargetProvider

	switch m.configDlg.Field {
	case views.FieldAIAPIKey:
		// After saving API key -> model selection (with saved custom models)
		cfg, err := config.New()
		customModels := []string{}
		if err == nil {
			customModels = cfg.GetCustomModels(provider)
		}
		m.configDlg.OpenAIProviderModel(provider, views.ProviderModelOptions[provider], customModels)

	case views.FieldAIModel:
		// After saving model -> endpoint
		pc := config.ProviderConfig{}
		for _, kp := range views.KnownProviders {
			if kp.Name == provider {
				pc.Endpoint = kp.DefaultEndpoint
				break
			}
		}
		m.configDlg.OpenAIProviderEndpoint(provider, pc.Endpoint)

	case views.FieldAIEndpoint:
		// Done configuring this provider -> back to provider list
		m.refreshProviderList()
	}
}

// refreshProviderList reloads the provider list from config.
func (m *Model) refreshProviderList() {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	// Set active provider to the one just configured
	provider := m.configDlg.TargetProvider
	if provider != "" {
		cfg.Set("ai.provider", provider)
		_ = cfg.Save()
	}

	m.configDlg.OpenAIProviderList(cfg.GetProviderKeys())
}

// saveGitHubToken handles GitHub token input.
func (m *Model) saveGitHubToken() {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	value := m.configDlg.GetInput()
	cfg.Set("github.token", value)
	if err := cfg.Save(); err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
		return
	}
	// Validate token by fetching user
	go m.validateGitHubToken(cfg, value)
}

// saveProviderAPIKey saves a provider's API key.
func (m *Model) saveProviderAPIKey() {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	provider := m.configDlg.TargetProvider
	value := m.configDlg.GetInput()
	pc := cfg.GetProviderConfig(provider)
	pc.APIKey = value
	cfg.SetProviderConfig(provider, pc)
	cfg.Set("ai.api_key", value) // also set top-level for backward compat
	cfg.Set("ai.provider", provider)

	if err := cfg.Save(); err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
		return
	}

	// Update aiData
	m.aiData.Provider = provider
	m.aiData.APIKey = value
	m.aiData.Sidebar.Provider = provider
	m.configDlg.SetResult(true, "saved")
}

// saveProviderModel saves the model for the target provider.
// If the model is not in the known options list, persists it as a custom model.
func (m *Model) saveProviderModel() {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	provider := m.configDlg.TargetProvider
	value := m.configDlg.GetInput()

	// Save model to provider config
	pc := cfg.GetProviderConfig(provider)
	pc.Model = value
	cfg.SetProviderConfig(provider, pc)
	cfg.Set("ai.model", value) // also top-level

	// Check if model is custom (not in known list) — persist it
	known := views.ProviderModelOptions[provider]
	isCustom := true
	for _, km := range known {
		if km == value {
			isCustom = false
			break
		}
	}
	if isCustom && len(known) > 0 {
		cfg.AddCustomModel(provider, value)
	}

	if err := cfg.Save(); err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
		return
	}

	m.aiData.Model = value
	m.aiData.Sidebar.Model = value
	m.configDlg.SetResult(true, "saved")
}

// saveProviderEndpoint saves the endpoint for the target provider.
func (m *Model) saveProviderEndpoint() {
	cfg, err := config.New()
	if err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Config error: %v", err))
		return
	}

	provider := m.configDlg.TargetProvider
	value := m.configDlg.GetInput()
	pc := cfg.GetProviderConfig(provider)
	pc.Endpoint = value
	cfg.SetProviderConfig(provider, pc)
	cfg.Set("ai.endpoint", value) // also top-level

	if err := cfg.Save(); err != nil {
		m.configDlg.SetResult(false, fmt.Sprintf("Save failed: %v", err))
		return
	}

	m.aiData.Endpoint = value
	m.configDlg.SetResult(true, "saved")
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
