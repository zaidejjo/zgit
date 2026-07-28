package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// ConfigState tracks the config dialog phase.
type ConfigState int

const (
	ConfigClosed  ConfigState = iota
	ConfigInput               // entering a value
	ConfigSelect              // selecting from a list
	ConfigLoading             // validating (e.g. fetching user from token)
	ConfigResult              // showing success/error
)

// ConfigField identifies which field is being edited.
type ConfigField int

const (
	FieldGitHubToken ConfigField = iota
	FieldAIProvider
	FieldAIModel
	FieldAIAPIKey
	FieldAIEndpoint
	FieldAIProviderMenu    // pick which provider to configure
	FieldAIProviderList    // select from known provider names
	FieldAIProviderActions // actions for an already-configured provider
)

// KnownProviders is the list of supported AI providers + their default models.
type KnownProvider struct {
	Name            string
	DefaultModel    string
	DefaultEndpoint string
}

// KnownProviders is ordered with featured providers first, then alphabetical extras.
// The provider picker uses this to show featured ones prominently.
var KnownProviders = []KnownProvider{
	{Name: "openrouter", DefaultModel: "openai/gpt-4o-mini", DefaultEndpoint: "https://openrouter.ai/api/v1/chat/completions"},
	{Name: "openai", DefaultModel: "gpt-4o-mini", DefaultEndpoint: "https://api.openai.com/v1/chat/completions"},
	{Name: "anthropic", DefaultModel: "claude-3-5-sonnet-latest", DefaultEndpoint: "https://api.anthropic.com/v1/messages"},
	{Name: "opencodezen", DefaultModel: "", DefaultEndpoint: ""},
	{Name: "deepseek", DefaultModel: "deepseek/deepseek-v4-flash", DefaultEndpoint: "https://api.deepseek.com/chat/completions"},
	{Name: "groq", DefaultModel: "llama-3.1-8b-instant", DefaultEndpoint: "https://api.groq.com/openai/v1/chat/completions"},
	{Name: "ollama", DefaultModel: "llama3.2", DefaultEndpoint: "http://localhost:11434/v1/chat/completions"},
	{Name: "custom", DefaultModel: "", DefaultEndpoint: ""},
}

// ProviderModels maps provider names to recommended/fetchable models.
var ProviderModelOptions = map[string][]string{
	"openrouter":  {"deepseek/deepseek-r1", "anthropic/claude-3.5-sonnet", "openai/gpt-4o", "google/gemini-2.5-flash"},
	"openai":      {"gpt-4o", "gpt-4o-mini", "o1", "o3-mini"},
	"anthropic":   {"claude-3-5-sonnet-latest", "claude-3-5-haiku-latest", "claude-3-opus-latest"},
	"opencodezen": nil, // routing provider — user types model ID manually
	"deepseek":    {"deepseek/deepseek-v4-flash", "deepseek/deepseek-v4-pro"},
	"groq":        {"llama-3.1-8b-instant", "llama-3.1-70b-versatile", "llama-3.3-70b-versatile", "mixtral-8x7b-32768", "gemma2-9b-it"},
	"ollama":      {"llama3.2", "llama3.1", "mistral", "codellama", "mixtral"},
	"custom":      nil,
}

// CustomModelOption is the sentinel label appended to model lists for custom entry.
const CustomModelOption = "+ Custom Model..."

// IsCustomModelOption returns true if the item is the custom model sentinel.
func IsCustomModelOption(item string) bool {
	return item == CustomModelOption
}

// ConfigModel is an overlay for editing config values with multi-step provider support.
type ConfigModel struct {
	State   ConfigState
	Field   ConfigField
	Input   string
	Cursor  int
	Result  string
	Success bool
	Width   int

	// List selection state
	SelectItems  []string // items for selection list
	SelectCursor int

	// Provider being configured (set when entering provider-specific fields)
	TargetProvider string
}

// NewConfigModel creates a default config dialog.
func NewConfigModel() ConfigModel {
	return ConfigModel{}
}

// OpenGitHubToken starts editing the GitHub token.
func (m *ConfigModel) OpenGitHubToken() {
	m.State = ConfigInput
	m.Field = FieldGitHubToken
	m.Input = ""
	m.Cursor = 0
	m.Result = ""
	m.Success = false
	m.TargetProvider = ""
}

// OpenAIProviderList shows the provider menu for selecting which to configure.
func (m *ConfigModel) OpenAIProviderList(configuredProviders []string) {
	// Build list: existing providers + "Add new provider"
	items := make([]string, 0, len(configuredProviders)+1)
	for _, p := range configuredProviders {
		items = append(items, "◎ "+p)
	}
	items = append(items, "＋ Add new provider")
	m.SelectItems = items
	m.SelectCursor = 0
	m.State = ConfigSelect
	m.Field = FieldAIProviderList
	m.Result = ""
	m.Success = false
}

// OpenAIProviderPicker shows the featured providers list + "Add custom provider...".
func (m *ConfigModel) OpenAIProviderPicker() {
	// First call: show featured 4 + custom option
	// Second call (from processConfigAction "Add custom provider..."): show full list
	m.SelectItems = []string{
		"openrouter",
		"openai",
		"anthropic",
		"opencodezen",
		"───",
		"Add custom provider...",
	}
	m.SelectCursor = 0
	m.State = ConfigSelect
	m.Field = FieldAIProviderMenu
	m.Result = ""
	m.Success = false
}

// OpenAIProviderFullList shows all known providers (for "Add custom provider..." flow).
func (m *ConfigModel) OpenAIProviderFullList() {
	items := make([]string, len(KnownProviders))
	for i, p := range KnownProviders {
		items[i] = p.Name
	}
	m.SelectItems = items
	m.SelectCursor = 0
	m.State = ConfigSelect
	m.Field = FieldAIProviderMenu
	m.Result = ""
	m.Success = false
}

// OpenAIProviderActions shows actions for an already-configured provider.
func (m *ConfigModel) OpenAIProviderActions(provider string) {
	m.TargetProvider = provider
	m.SelectItems = []string{
		"Edit API Key",
		"Change Model",
		"Change Endpoint",
		"───",
		"Done",
	}
	m.SelectCursor = 0
	m.State = ConfigSelect
	m.Field = FieldAIProviderActions
	m.Result = ""
	m.Success = false
}

// OpenAIProviderKey starts editing the API key for a specific provider.
func (m *ConfigModel) OpenAIProviderKey(provider string) {
	m.State = ConfigInput
	m.Field = FieldAIAPIKey
	m.TargetProvider = provider
	m.Input = ""
	m.Cursor = 0
	m.Result = ""
	m.Success = false
}

// OpenAIProviderModel opens model selection for a provider.
// Merges static known models + saved custom models, appends "+ Custom Model..." at end.
func (m *ConfigModel) OpenAIProviderModel(provider string, options []string, customModels []string) {
	// Build merged list: known options + saved custom models
	merged := make([]string, 0, len(options)+len(customModels)+1)
	merged = append(merged, options...)

	// Append any saved custom models not already in the known list
	known := make(map[string]bool, len(options))
	for _, o := range options {
		known[o] = true
	}
	for _, cm := range customModels {
		if !known[cm] {
			merged = append(merged, cm)
		}
	}

	if len(merged) == 0 && len(customModels) == 0 {
		// No known options and no saved custom models — free-form input
		m.State = ConfigInput
		m.Field = FieldAIModel
		m.TargetProvider = provider
		m.Input = ""
		m.Cursor = 0
		m.Result = ""
		m.Success = false
		return
	}

	// Append "+ Custom Model..." at the end
	merged = append(merged, CustomModelOption)
	m.SelectItems = merged
	m.SelectCursor = 0
	m.State = ConfigSelect
	m.Field = FieldAIModel
	m.TargetProvider = provider
	m.Result = ""
	m.Success = false
}

// OpenAIProviderEndpoint opens endpoint editing for a provider.
func (m *ConfigModel) OpenAIProviderEndpoint(provider string, defaultEndpoint string) {
	m.State = ConfigInput
	m.Field = FieldAIEndpoint
	m.TargetProvider = provider
	m.Input = defaultEndpoint
	m.Cursor = len(m.Input)
	m.Result = ""
	m.Success = false
}

// Active returns true if dialog is visible.
func (m *ConfigModel) Active() bool {
	return m.State != ConfigClosed
}

// HandlePaste inserts pasted text at cursor position. Called from app level on tea.PasteMsg.
func (m *ConfigModel) HandlePaste(text string) {
	if m.State != ConfigInput {
		return
	}
	// Insert text at cursor, preserving masked display
	m.Input = m.Input[:m.Cursor] + text + m.Input[m.Cursor:]
	m.Cursor += len(text)
}

// Close dismisses the dialog.
func (m *ConfigModel) Close() {
	m.State = ConfigClosed
	m.TargetProvider = ""
	m.SelectItems = nil
}

// GetInput returns trimmed input.
func (m *ConfigModel) GetInput() string {
	return strings.TrimSpace(m.Input)
}

// GetSelectedItem returns the currently selected list item.
func (m *ConfigModel) GetSelectedItem() string {
	if m.SelectCursor >= 0 && m.SelectCursor < len(m.SelectItems) {
		return m.SelectItems[m.SelectCursor]
	}
	return ""
}

// GetSelectedProvider extracts provider name from list item (remove prefix icons).
func (m *ConfigModel) GetSelectedProvider() string {
	item := m.GetSelectedItem()
	item = strings.TrimPrefix(item, "◎ ")
	item = strings.TrimPrefix(item, "＋ ")
	return item
}

// HandleKey processes key presses. Returns true if handled.
func (m *ConfigModel) HandleKey(key string) bool {
	switch m.State {
	case ConfigInput:
		return m.handleInputKey(key)
	case ConfigSelect:
		return m.handleSelectKey(key)
	case ConfigLoading:
		return true
	case ConfigResult:
		switch key {
		case "esc", "enter":
			m.Close()
			return true
		}
	}
	return true
}

func (m *ConfigModel) handleInputKey(key string) bool {
	switch key {
	case "esc":
		// Go back to provider list if configuring a provider
		if m.TargetProvider != "" && m.Field != FieldGitHubToken {
			// Signal to parent to go back to provider list
			m.State = ConfigResult
			m.Success = true
			m.Result = "saved"
			return true
		}
		m.Close()
		return true
	case "enter":
		if m.GetInput() != "" || m.Field == FieldAIEndpoint {
			m.State = ConfigLoading
			return true
		}
		return true
	case "backspace":
		if m.Cursor > 0 {
			m.Input = m.Input[:m.Cursor-1] + m.Input[m.Cursor:]
			m.Cursor--
		}
	case "left":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "right":
		if m.Cursor < len(m.Input) {
			m.Cursor++
		}
	case "home":
		m.Cursor = 0
	case "end":
		m.Cursor = len(m.Input)
	case " ":
		m.Input = m.Input[:m.Cursor] + " " + m.Input[m.Cursor:]
		m.Cursor++
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.Input = m.Input[:m.Cursor] + key + m.Input[m.Cursor:]
			m.Cursor++
		}
	}
	return true
}

func (m *ConfigModel) handleSelectKey(key string) bool {
	switch key {
	case "esc":
		m.Close()
		return true
	case "enter":
		if m.SelectCursor >= 0 && m.SelectCursor < len(m.SelectItems) {
			m.State = ConfigLoading
			return true
		}
		return true
	case "j", "down":
		if m.SelectCursor < len(m.SelectItems)-1 {
			m.SelectCursor++
		}
	case "k", "up":
		if m.SelectCursor > 0 {
			m.SelectCursor--
		}
	case "g", "home":
		m.SelectCursor = 0
	case "G", "end":
		m.SelectCursor = len(m.SelectItems) - 1
	}
	return true
}

// SetResult displays the outcome.
func (m *ConfigModel) SetResult(success bool, msg string) {
	m.Success = success
	m.Result = msg
	m.State = ConfigResult
}

// View renders the config overlay.
func (m *ConfigModel) View(width int) string {
	if !m.Active() {
		return ""
	}

	var b strings.Builder
	dialogW := width - 8
	if dialogW < 30 {
		dialogW = 30
	}
	if dialogW > 72 {
		dialogW = 72
	}

	title := m.dialogTitle()

	switch m.State {
	case ConfigInput:
		m.renderInput(&b, dialogW, title)
	case ConfigSelect:
		m.renderSelect(&b, dialogW, title)
	case ConfigLoading:
		m.renderLoading(&b, dialogW, title)
	case ConfigResult:
		m.renderResult(&b, dialogW, title)
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(b.String())
}

func (m *ConfigModel) dialogTitle() string {
	prefix := "⚙"
	switch m.Field {
	case FieldGitHubToken:
		return prefix + " GitHub Token"
	case FieldAIProvider, FieldAIProviderMenu, FieldAIProviderList, FieldAIProviderActions:
		if m.Field == FieldAIProviderActions && m.TargetProvider != "" {
			return prefix + " " + m.TargetProvider
		}
		return prefix + " AI Providers"
	case FieldAIAPIKey:
		if m.TargetProvider != "" {
			return prefix + " " + m.TargetProvider + " API Key"
		}
		return prefix + " AI API Key"
	case FieldAIModel:
		if m.TargetProvider != "" {
			return prefix + " " + m.TargetProvider + " Model"
		}
		return prefix + " AI Model"
	case FieldAIEndpoint:
		if m.TargetProvider != "" {
			return prefix + " " + m.TargetProvider + " Endpoint"
		}
		return prefix + " AI Endpoint"
	}
	return prefix + " Config"
}

func (m *ConfigModel) renderInput(b *strings.Builder, dialogW int, title string) {
	prompt := ""
	switch m.Field {
	case FieldGitHubToken:
		prompt = "Enter GitHub Personal Access Token (classic or fine-grained):"
	case FieldAIAPIKey:
		prompt = "Enter API key:"
	case FieldAIModel:
		prompt = "Enter model name:"
	case FieldAIEndpoint:
		prompt = "Enter API endpoint URL:"
	}

	display := m.Input
	// Mask token/API key fields
	if m.Field == FieldGitHubToken || m.Field == FieldAIAPIKey {
		display = strings.Repeat("•", len(m.Input))
		if m.Cursor < len(display) {
			display = display[:m.Cursor] + "█" + display[m.Cursor:]
		}
	} else if m.Cursor < len(display) {
		display = display[:m.Cursor] + "█" + display[m.Cursor:]
	} else {
		display += " "
	}

	inputLine := "> " + display

	hint := " Enter to confirm  Esc to cancel"
	if m.TargetProvider != "" {
		hint = " Enter to confirm  Esc to go back"
	}

	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(dialogW).
		Padding(0, 1).
		Background(styles.Surface).
		Render(strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
			"",
			lipgloss.NewStyle().Foreground(styles.Subtext).Render(prompt),
			inputLine,
			"",
			lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(hint),
		}, "\n")))
}

func (m *ConfigModel) renderSelect(b *strings.Builder, dialogW int, title string) {
	items := make([]string, 0, len(m.SelectItems)+4)
	items = append(items, lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title))
	items = append(items, "")
	items = append(items, lipgloss.NewStyle().Foreground(styles.Subtext).Render(" Select an option:"))

	avail := 12 // header + hints
	maxItems := dialogW/2 - 2
	if maxItems < 5 {
		maxItems = 5
	}
	start := 0
	if m.SelectCursor > maxItems-1 {
		start = m.SelectCursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(m.SelectItems) {
		end = len(m.SelectItems)
	}

	for i := start; i < end; i++ {
		item := m.SelectItems[i]
		if i == m.SelectCursor {
			items = append(items, lipgloss.NewStyle().
				Foreground(styles.Mauve).
				Bold(true).
				Render("▸ "+item))
		} else {
			items = append(items, lipgloss.NewStyle().
				Foreground(styles.Text).
				Render("  "+item))
		}
		avail++
	}

	items = append(items, "")
	hint := " ↑↓ navigate  Enter select  Esc cancel"
	if m.TargetProvider != "" {
		hint = " ↑↓ navigate  Enter select"
	}
	items = append(items, lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(hint))

	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(dialogW).
		Padding(0, 1).
		Background(styles.Surface).
		Render(strings.Join(items, "\n")))
}

func (m *ConfigModel) renderLoading(b *strings.Builder, dialogW int, title string) {
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(dialogW).
		Padding(0, 1).
		Background(styles.Surface).
		Render(strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
			"",
			styles.LoadingStyle.Render("Saving..."),
		}, "\n")))
}

func (m *ConfigModel) renderResult(b *strings.Builder, dialogW int, title string) {
	result := m.Result
	if m.Success {
		result = styles.StatusStagedStyle.Render("✓ " + m.Result)
	} else {
		result = styles.ErrorStyle.Render("✗ " + m.Result)
	}

	hint := " Enter or Esc to close"
	if m.TargetProvider != "" && m.Result == "saved" {
		hint = " Enter or Esc to continue configuring"
	}

	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(dialogW).
		Padding(0, 1).
		Background(styles.Surface).
		Render(strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
			"",
			result,
			"",
			lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(hint),
		}, "\n")))
}

// FieldLabel returns a human-readable label for the config field.
func (f ConfigField) FieldLabel() string {
	switch f {
	case FieldGitHubToken:
		return "github.token"
	case FieldAIProvider:
		return "ai.provider"
	case FieldAIModel:
		return "ai.model"
	case FieldAIAPIKey:
		return "ai.api_key"
	case FieldAIEndpoint:
		return "ai.endpoint"
	case FieldAIProviderMenu:
		return "ai.provider_menu"
	case FieldAIProviderList:
		return "ai.provider_list"
	case FieldAIProviderActions:
		return "ai.provider_actions"
	}
	return fmt.Sprintf("unknown(%d)", f)
}
