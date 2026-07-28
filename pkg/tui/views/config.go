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
)

// ConfigModel is a lightweight overlay for editing config values.
type ConfigModel struct {
	State   ConfigState
	Field   ConfigField
	Input   string
	Cursor  int
	Result  string
	Success bool
	Width   int
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
}

// OpenAI starts editing AI settings.
func (m *ConfigModel) OpenAI() {
	m.State = ConfigInput
	m.Field = FieldAIProvider
	m.Input = ""
	m.Cursor = 0
	m.Result = ""
	m.Success = false
}

// Active returns true if dialog is visible.
func (m *ConfigModel) Active() bool {
	return m.State != ConfigClosed
}

// Close dismisses the dialog.
func (m *ConfigModel) Close() {
	m.State = ConfigClosed
}

// GetInput returns trimmed input.
func (m *ConfigModel) GetInput() string {
	return strings.TrimSpace(m.Input)
}

// HandleKey processes key presses. Returns true if handled.
func (m *ConfigModel) HandleKey(key string) bool {
	switch m.State {
	case ConfigInput:
		switch key {
		case "esc":
			m.Close()
			return true
		case "enter":
			if m.GetInput() != "" {
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

	title := ""
	switch m.Field {
	case FieldGitHubToken:
		title = "⚙ GitHub Token"
	case FieldAIProvider:
		title = "⚙ AI Provider"
	case FieldAIModel:
		title = "⚙ AI Model"
	case FieldAIAPIKey:
		title = "⚙ AI API Key"
	}

	switch m.State {
	case ConfigInput:
		prompt := ""
		switch m.Field {
		case FieldGitHubToken:
			prompt = "Enter GitHub Personal Access Token (classic or fine-grained):"
		case FieldAIProvider:
			prompt = "Enter AI provider (openai, anthropic, deepseek, openrouter):"
		case FieldAIModel:
			prompt = "Enter model name (e.g. gpt-4o, claude-sonnet-4-20250514):"
		case FieldAIAPIKey:
			prompt = "Enter API key:"
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
				lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Enter to confirm  Esc to cancel"),
			}, "\n")))

	case ConfigLoading:
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Mauve).
			Width(dialogW).
			Padding(0, 1).
			Background(styles.Surface).
			Render(strings.Join([]string{
				lipgloss.NewStyle().Bold(true).Foreground(styles.Mauve).Render(title),
				"",
				styles.LoadingStyle.Render("Validating..."),
			}, "\n")))

	case ConfigResult:
		result := m.Result
		if m.Success {
			result = styles.StatusStagedStyle.Render("✓ " + m.Result)
		} else {
			result = styles.ErrorStyle.Render("✗ " + m.Result)
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
				lipgloss.NewStyle().Foreground(styles.Subtext).Italic(true).Render(" Enter or Esc to close"),
			}, "\n")))
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(b.String())
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
	}
	return fmt.Sprintf("unknown(%d)", f)
}
