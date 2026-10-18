package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// ConfigState tracks the config dialog phase.
type ConfigState int

const (
	ConfigClosed  ConfigState = iota
	ConfigInput               // entering a value (huh input form)
	ConfigSelect              // selecting from a list (huh select form)
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

// ProviderModelOptions maps provider names to recommended/fetchable models.
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
// Each step builds a single-field huh form; the app orchestrator (app_config.go)
// advances steps exactly as before (completion == old Enter semantics).
type ConfigModel struct {
	State   ConfigState
	Field   ConfigField
	Input   string // last completed input value (GetInput compat)
	Cursor  int
	Result  string
	Success bool
	Width   int

	// List selection state (SelectItems builds the form; SelectCursor is
	// synced on completion so GetSelectedItem keeps working).
	SelectItems  []string // items for selection list
	SelectCursor int

	// Provider being configured (set when entering provider-specific fields)
	TargetProvider string

	form      *huh.Form
	inputVal  string // huh-bound input value
	selectVal string // huh-bound select value
	inputMode bool   // true when the current step is a text input
	needInit  bool
}

// NewConfigModel creates a default config dialog.
func NewConfigModel() ConfigModel {
	return ConfigModel{}
}

// buildInput creates the text-entry form for the current Field.
func (m *ConfigModel) buildInput(prompt, placeholder string, password, allowEmpty bool) {
	m.inputMode = true
	field := huh.NewInput().
		Title(prompt).
		Placeholder(placeholder).
		Value(&m.inputVal)
	if password {
		field = field.EchoMode(huh.EchoModePassword)
	}
	if !allowEmpty {
		field = field.Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("a value is required")
			}
			return nil
		})
	}
	m.form = huh.NewForm(huh.NewGroup(field)).WithShowHelp(true)
	m.applyWidth()
	m.needInit = true
}

// buildSelect creates the list form from SelectItems (values == labels so
// GetSelectedItem keeps returning the exact legacy strings).
func (m *ConfigModel) buildSelect(title string) {
	m.inputMode = false
	opts := make([]huh.Option[string], 0, len(m.SelectItems))
	for _, item := range m.SelectItems {
		opts = append(opts, huh.NewOption(item, item))
	}
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Key("pick").
				Options(opts...).
				Value(&m.selectVal),
		),
	).WithShowHelp(true)
	m.applyWidth()
	m.needInit = true
}

func (m *ConfigModel) applyWidth() {
	if m.form != nil && m.Width > 0 {
		m.form = m.form.WithWidth(m.Width - 4)
	}
}

// OpenGitHubToken starts editing the GitHub token.
func (m *ConfigModel) OpenGitHubToken() {
	m.State = ConfigInput
	m.Field = FieldGitHubToken
	m.inputVal = ""
	m.Input = ""
	m.Result = ""
	m.Success = false
	m.TargetProvider = ""
	m.buildInput("GitHub Token", "ghp_… / github_pat_…", true, false)
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
	m.buildSelect("AI Providers")
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
	m.buildSelect("Pick a provider")
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
	m.buildSelect("Pick a provider")
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
	m.buildSelect(provider)
}

// OpenAIProviderKey starts editing the API key for a specific provider.
func (m *ConfigModel) OpenAIProviderKey(provider string) {
	m.State = ConfigInput
	m.Field = FieldAIAPIKey
	m.TargetProvider = provider
	m.inputVal = ""
	m.Input = ""
	m.Result = ""
	m.Success = false
	m.buildInput(provider+" API Key", "sk-…", true, false)
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
		m.openCustomModelInput(provider)
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
	m.buildSelect(provider + " Model")
}

// OpenCustomModelInput opens free-form model entry (custom model flow).
func (m *ConfigModel) OpenCustomModelInput(provider string) {
	m.openCustomModelInput(provider)
}

func (m *ConfigModel) openCustomModelInput(provider string) {
	m.State = ConfigInput
	m.Field = FieldAIModel
	m.TargetProvider = provider
	m.inputVal = ""
	m.Input = ""
	m.Result = ""
	m.Success = false
	m.buildInput(provider+" Model", "model id", false, false)
}

// OpenAIProviderEndpoint opens endpoint editing for a provider.
func (m *ConfigModel) OpenAIProviderEndpoint(provider string, defaultEndpoint string) {
	m.State = ConfigInput
	m.Field = FieldAIEndpoint
	m.TargetProvider = provider
	m.inputVal = defaultEndpoint
	m.Input = defaultEndpoint
	m.Result = ""
	m.Success = false
	m.buildInput(provider+" Endpoint", "https://…", false, true)
}

// Active returns true if dialog is visible.
func (m *ConfigModel) Active() bool {
	return m.State != ConfigClosed
}

// HandlePaste appends pasted text to the input value. Called from app level
// on tea paste messages.
func (m *ConfigModel) HandlePaste(text string) {
	if m.State != ConfigInput || !m.inputMode {
		return
	}
	m.inputVal += text
}

// Close dismisses the dialog.
func (m *ConfigModel) Close() {
	m.State = ConfigClosed
	m.TargetProvider = ""
	m.SelectItems = nil
	m.form = nil
}

// GetInput returns trimmed input.
func (m *ConfigModel) GetInput() string {
	return strings.TrimSpace(m.Input)
}

// GetSelectedItem returns the currently selected list item.
func (m *ConfigModel) GetSelectedItem() string {
	if m.selectVal != "" {
		return m.selectVal
	}
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

// Update drives the huh form. Completion maps to ConfigLoading (the app
// orchestrator picks it up exactly like the old Enter key did).
func (m *ConfigModel) Update(msg tea.Msg) (*ConfigModel, tea.Cmd) {
	if !m.Active() {
		return m, nil
	}
	if m.needInit {
		m.needInit = false
		if m.form != nil {
			return m, m.form.Init()
		}
		return m, nil
	}

	// Loading and Result states swallow input except close keys.
	if m.State == ConfigLoading {
		return m, nil
	}
	if m.State == ConfigResult {
		if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "esc" || key.String() == "enter") {
			m.Close()
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.applyWidth()
		return m, nil

	case tea.KeyMsg:
		// Esc cancels — or steps back to the provider list mid-flow
		// (preserves the legacy "Esc to go back" signal).
		if msg.String() == "esc" {
			if m.inputMode && m.TargetProvider != "" && m.Field != FieldGitHubToken {
				m.State = ConfigResult
				m.Success = true
				m.Result = "saved"
				return m, nil
			}
			m.Close()
			return m, nil
		}
	}

	if m.form == nil {
		return m, nil
	}
	f, cmd := m.form.Update(msg)
	if nf, ok := f.(*huh.Form); ok {
		m.form = nf
	}
	if m.form.State == huh.StateCompleted {
		m.completeStep()
		return m, nil
	}
	if m.form.State == huh.StateAborted {
		m.Close()
		return m, nil
	}
	return m, cmd
}

// RefreshForm rebuilds the current step's form (used when staying on a
// step, e.g. separator rows that complete without choosing).
func (m *ConfigModel) RefreshForm() {
	switch {
	case m.inputMode:
		m.buildInput(m.inputTitle(), m.inputPlaceholder(), m.inputPassword(), m.inputAllowEmpty())
	default:
		m.buildSelect(m.selectTitle())
	}
}

func (m *ConfigModel) inputTitle() string {
	switch m.Field {
	case FieldGitHubToken:
		return "GitHub Token"
	case FieldAIAPIKey:
		return m.TargetProvider + " API Key"
	case FieldAIModel:
		return m.TargetProvider + " Model"
	default:
		return m.TargetProvider + " Endpoint"
	}
}

func (m *ConfigModel) inputPlaceholder() string {
	switch m.Field {
	case FieldGitHubToken:
		return "ghp_… / github_pat_…"
	case FieldAIAPIKey:
		return "sk-…"
	case FieldAIModel:
		return "model id"
	default:
		return "https://…"
	}
}

func (m *ConfigModel) inputPassword() bool {
	return m.Field == FieldGitHubToken || m.Field == FieldAIAPIKey
}

func (m *ConfigModel) inputAllowEmpty() bool {
	return m.Field == FieldAIEndpoint
}

func (m *ConfigModel) selectTitle() string {
	switch m.Field {
	case FieldAIProviderList:
		return "AI Providers"
	case FieldAIProviderMenu:
		return "Pick a provider"
	case FieldAIProviderActions:
		return m.TargetProvider
	default:
		return m.TargetProvider + " Model"
	}
}

// completeStep syncs form values and advances to ConfigLoading so the
// orchestrator processes the step (same as the legacy Enter key).
func (m *ConfigModel) completeStep() {
	if m.inputMode {
		m.Input = m.inputVal
	} else {
		m.Input = ""
		for i, item := range m.SelectItems {
			if item == m.selectVal {
				m.SelectCursor = i
				break
			}
		}
	}
	m.State = ConfigLoading
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

	// Total box width target; content inside is 4 cells narrower because
	// lipgloss draws the border and padding outside Style.Width.
	boxW := width - dialogMargin
	if boxW > 76 {
		boxW = 76
	}
	if boxW < 34 {
		boxW = 34
	}
	contentW := contentWidth(boxW)

	title := m.dialogTitle()

	switch m.State {
	case ConfigInput, ConfigSelect:
		if m.form != nil {
			m.form = m.form.WithWidth(contentW - formFieldFrame)
		}
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Mauve).
			Width(styleWidth(boxW)).
			Padding(0, 1).
			Background(styles.Surface).
			Render(clampBlock(strings.Join([]string{
				lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
				"",
				m.formView(),
			}, "\n"), contentW)))
	case ConfigLoading:
		m.renderLoading(&b, boxW, title)
	case ConfigResult:
		m.renderResult(&b, boxW, title)
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(b.String())
}

// formView renders the huh form (guarded for nil during transitions).
func (m *ConfigModel) formView() string {
	if m.form == nil {
		return styles.LoadingStyle.Render("Loading...")
	}
	return m.form.View()
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

func (m *ConfigModel) renderLoading(b *strings.Builder, boxW int, title string) {
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Width(styleWidth(boxW)).
		Padding(0, 1).
		Background(styles.Surface).
		Render(clampBlock(strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
			"",
			styles.LoadingStyle.Render("Saving..."),
		}, "\n"), contentWidth(boxW))))
}

func (m *ConfigModel) renderResult(b *strings.Builder, boxW int, title string) {
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
		Width(styleWidth(boxW)).
		Padding(0, 1).
		Background(styles.Surface).
		Render(clampBlock(strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
			"",
			result,
			"",
			lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(hint),
		}, "\n"), contentWidth(boxW))))
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
