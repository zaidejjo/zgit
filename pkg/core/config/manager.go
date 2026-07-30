// Package config manages zgit configuration via a YAML file.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const configFileName = "config.yaml"

// Config holds all zgit configuration values.
type Config struct {
	GitHub          GitHubConfig    `mapstructure:"github"`
	AI              AIConfig        `mapstructure:"ai"`
	Repo            RepoConfig      `mapstructure:"repo"`
	RecentRepos     []string        `mapstructure:"recent_repos"`
	Theme           string          `mapstructure:"theme"`
	UserPreferences UserPreferences `mapstructure:"user_preferences"`
}

// UserPreferences stores user-customizable appearance and keybinding settings.
type UserPreferences struct {
	Appearance  AppearanceConfig  `mapstructure:"appearance"`
	Keybindings map[string]string `mapstructure:"keybindings"`
}

// AppearanceConfig controls theme-level visual customization.
type AppearanceConfig struct {
	Theme       string `mapstructure:"theme"`
	AccentColor string `mapstructure:"accent_color"`
	Brightness  int    `mapstructure:"brightness"`
}

// ProviderConfig stores per-provider AI settings.
type ProviderConfig struct {
	APIKey       string   `mapstructure:"api_key"`
	Model        string   `mapstructure:"model"`
	Endpoint     string   `mapstructure:"endpoint,omitempty"`
	CustomModels []string `mapstructure:"custom_models,omitempty"`
}

// AIConfig stores AI commit message provider settings.
type AIConfig struct {
	Provider  string                    `mapstructure:"provider"`
	APIKey    string                    `mapstructure:"api_key"`
	Model     string                    `mapstructure:"model"`
	Endpoint  string                    `mapstructure:"endpoint"`
	Providers map[string]ProviderConfig `mapstructure:"providers,omitempty"`
}

// GitHubConfig stores GitHub authentication and preferences.
type GitHubConfig struct {
	Token string `mapstructure:"token"`
	User  string `mapstructure:"user"`
}

// RepoConfig stores default repository behavior.
type RepoConfig struct {
	Editor      string   `mapstructure:"editor"`
	DiffContext int      `mapstructure:"diff_context"`
	WatchPaths  []string `mapstructure:"watch_paths"`
}

// Manager loads, saves, and accesses zgit configuration.
type Manager struct {
	v    *viper.Viper
	path string // config directory
}

// New creates a ConfigManager, reading config from the default path (~/.config/zgit/).
func New() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	configDir := filepath.Join(home, ".config", "zgit")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Defaults
	v.SetDefault("github.user", "")
	v.SetDefault("github.token", "")
	v.SetDefault("ai.provider", "")
	v.SetDefault("ai.api_key", "")
	v.SetDefault("ai.model", "")
	v.SetDefault("ai.endpoint", "")
	v.SetDefault("ai.providers", map[string]interface{}{})
	v.SetDefault("repo.editor", "code")
	v.SetDefault("repo.diff_context", 3)
	v.SetDefault("theme", "default")
	v.SetDefault("user_preferences.appearance.theme", "dark")
	v.SetDefault("user_preferences.appearance.accent_color", "")
	v.SetDefault("user_preferences.appearance.brightness", 50)
	v.SetDefault("user_preferences.keybindings", map[string]interface{}{
		"command_palette": "Ctrl+K",
		"toggle_ai_panel": "Ctrl+Shift+A",
		"fullscreen_ai":   "Ctrl+Shift+F",
		"commit":          "Enter",
		"close_dialog":    "Escape",
	})

	m := &Manager{v: v, path: configDir}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// First run — create default config
			if err := m.Save(); err != nil {
				return nil, fmt.Errorf("save default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return m, nil
}

// Load returns the full parsed config.
func (m *Manager) Load() (*Config, error) {
	var cfg Config
	if err := m.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// Save writes current config to disk.
func (m *Manager) Save() error {
	configFile := filepath.Join(m.path, configFileName)
	return m.v.WriteConfigAs(configFile)
}

// Set sets a config key. Use dot notation (e.g. "github.token").
func (m *Manager) Set(key string, value interface{}) {
	m.v.Set(key, value)
}

// GetString returns a string config value.
func (m *Manager) GetString(key string) string {
	return m.v.GetString(key)
}

// GetRecentRepos returns the list of recently opened repositories.
func (m *Manager) GetRecentRepos() []string {
	return m.v.GetStringSlice("recent_repos")
}

// AddRecentRepo adds a path to the recent repos list (max 20, deduped).
func (m *Manager) AddRecentRepo(path string) {
	repos := m.GetRecentRepos()
	// Remove existing entry (dedupe)
	filtered := make([]string, 0, len(repos))
	for _, r := range repos {
		if r != path {
			filtered = append(filtered, r)
		}
	}
	// Prepend
	filtered = append([]string{path}, filtered...)
	// Trim to 20
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}
	m.v.Set("recent_repos", filtered)
}

// ClearRecentRepos removes all recent repos.
func (m *Manager) ClearRecentRepos() {
	m.v.Set("recent_repos", []string{})
}

// ConfigPath returns the config directory path.
func (m *Manager) ConfigPath() string {
	return m.path
}

// ConfigFilePath returns the full path to the config file.
func (m *Manager) ConfigFilePath() string {
	return filepath.Join(m.path, configFileName)
}

// GetInt returns an int config value.
func (m *Manager) GetInt(key string) int {
	return m.v.GetInt(key)
}

// GetBool returns a bool config value.
func (m *Manager) GetBool(key string) bool {
	return m.v.GetBool(key)
}

// GetProviderKeys returns all configured provider names.
func (m *Manager) GetProviderKeys() []string {
	providers := m.v.GetStringMap("ai.providers")
	keys := make([]string, 0, len(providers))
	for k := range providers {
		keys = append(keys, k)
	}
	return keys
}

// GetProviderConfig returns config for a specific provider, falling back to top-level defaults.
func (m *Manager) GetProviderConfig(provider string) ProviderConfig {
	key := "ai.providers." + provider
	pc := ProviderConfig{
		APIKey:   m.v.GetString(key + ".api_key"),
		Model:    m.v.GetString(key + ".model"),
		Endpoint: m.v.GetString(key + ".endpoint"),
	}
	// Fall back to top-level if per-provider not set
	if pc.APIKey == "" {
		pc.APIKey = m.GetString("ai.api_key")
	}
	if pc.Model == "" {
		pc.Model = m.GetString("ai.model")
	}
	if pc.Endpoint == "" {
		pc.Endpoint = m.GetString("ai.endpoint")
	}
	return pc
}

// SetProviderConfig saves config for a specific provider.
func (m *Manager) SetProviderConfig(provider string, pc ProviderConfig) {
	key := "ai.providers." + provider
	m.v.Set(key+".api_key", pc.APIKey)
	m.v.Set(key+".model", pc.Model)
	if pc.Endpoint != "" {
		m.v.Set(key+".endpoint", pc.Endpoint)
	}
	if len(pc.CustomModels) > 0 {
		m.v.Set(key+".custom_models", pc.CustomModels)
	}
}

// GetCustomModels returns saved custom models for a provider.
func (m *Manager) GetCustomModels(provider string) []string {
	key := "ai.providers." + provider + ".custom_models"
	return m.v.GetStringSlice(key)
}

// AddCustomModel adds a model to the provider's custom models list (deduped).
func (m *Manager) AddCustomModel(provider string, model string) {
	models := m.GetCustomModels(provider)
	for _, m := range models {
		if m == model {
			return // already exists
		}
	}
	models = append(models, model)
	m.v.Set("ai.providers."+provider+".custom_models", models)
}

// GetUserPreferences returns the stored user preferences.
func (m *Manager) GetUserPreferences() UserPreferences {
	return UserPreferences{
		Appearance: AppearanceConfig{
			Theme:       m.v.GetString("user_preferences.appearance.theme"),
			AccentColor: m.v.GetString("user_preferences.appearance.accent_color"),
			Brightness:  m.v.GetInt("user_preferences.appearance.brightness"),
		},
		Keybindings: m.v.GetStringMapString("user_preferences.keybindings"),
	}
}

// SetUserPreferences saves user preferences and persists to disk.
func (m *Manager) SetUserPreferences(prefs UserPreferences) error {
	m.v.Set("user_preferences.appearance.theme", prefs.Appearance.Theme)
	m.v.Set("user_preferences.appearance.accent_color", prefs.Appearance.AccentColor)
	m.v.Set("user_preferences.appearance.brightness", prefs.Appearance.Brightness)
	m.v.Set("user_preferences.keybindings", prefs.Keybindings)
	return m.Save()
}

// DeleteProviderConfig removes a provider's config.
func (m *Manager) DeleteProviderConfig(provider string) {
	key := "ai.providers." + provider
	m.v.Set(key, nil)
}
