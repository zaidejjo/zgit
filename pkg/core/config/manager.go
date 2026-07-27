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
	GitHub      GitHubConfig `mapstructure:"github"`
	AI          AIConfig     `mapstructure:"ai"`
	Repo        RepoConfig   `mapstructure:"repo"`
	RecentRepos []string     `mapstructure:"recent_repos"`
	Theme       string       `mapstructure:"theme"`
}

// AIConfig stores AI commit message provider settings.
type AIConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"`
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
	v.SetDefault("repo.editor", "code")
	v.SetDefault("repo.diff_context", 3)
	v.SetDefault("theme", "default")

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
