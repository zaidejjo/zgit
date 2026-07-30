// Package core provides the unified engine that composes all backend components.
// It is the single entry point for both TUI and Desktop presentation layers.
package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/ai"
	"github.com/zaidejjo/zgit/pkg/core/cache"
	"github.com/zaidejjo/zgit/pkg/core/config"
	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// EngineOptions configures engine behavior.
type EngineOptions struct {
	// RepoPath for local git operations. Empty = auto-detect from cwd.
	RepoPath string

	// CacheSize is the max number of cached API responses.
	CacheSize int

	// CacheTTL is how long cached items remain valid.
	CacheTTL int // seconds (0 = no expiration)
}

// DefaultEngineOptions returns sensible defaults.
func DefaultEngineOptions() EngineOptions {
	return EngineOptions{
		RepoPath:  "",
		CacheSize: 200,
		CacheTTL:  30, // 30 seconds
	}
}

// Engine is the top-level API consumed by presentation layers.
// It composes the git adapter, GitHub client, config manager, and cache.
type Engine struct {
	Git    gitpkg.GitAdapter
	GitHub github.GitHubClient
	Config *config.Manager
	Cache  *cache.Cache

	opts EngineOptions
	mu   sync.Mutex
	repo *models.Repo
}

// New creates a fully wired Engine.
func New(opts EngineOptions) (*Engine, error) {
	if opts.CacheSize <= 0 {
		opts.CacheSize = DefaultEngineOptions().CacheSize
	}

	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("init config: %w", err)
	}

	c, err := cache.New(opts.CacheSize, secondsToDuration(opts.CacheTTL))
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}

	git, err := gitpkg.NewNativeExec("")
	if err != nil {
		return nil, fmt.Errorf("init git: %w", err)
	}

	e := &Engine{
		Git:    git,
		Config: cfg,
		Cache:  c,
		opts:   opts,
	}

	// Set up GitHub client if token exists
	token := cfg.GetString("github.token")
	if token != "" {
		gh, err := github.NewCombinedClient(token)
		if err != nil {
			// Non-fatal: user can authenticate later
			fmt.Printf("warning: github client init: %v\n", err)
		} else {
			e.GitHub = gh
		}
	}

	return e, nil
}

// OpenRepo opens a local git repository. If repoPath is empty, detects from cwd.
func (e *Engine) OpenRepo(repoPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if repoPath == "" {
		repoPath = e.opts.RepoPath
	}
	if repoPath == "" {
		repoPath = "."
	}

	if err := e.Git.Open(repoPath); err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	// Try to detect GitHub repo from remotes
	if e.GitHub != nil {
		owner, name, err := guessCurrentRepo(e.Git)
		if err == nil {
			repo, err := e.GitHub.GetRepository(context.Background(), owner, name)
			if err == nil {
				e.repo = repo
				repo.Path = repoPath
				return nil
			}
		}
	}

	// Fallback: just set path
	e.repo = &models.Repo{Path: repoPath}
	return nil
}

// CloseRepo closes the current repository.
func (e *Engine) CloseRepo() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.repo = nil
	return e.Git.Close()
}

// CurrentRepo returns the currently open repo metadata.
func (e *Engine) CurrentRepo() *models.Repo {
	return e.repo
}

// AuthenticateGitHub sets the GitHub token and initializes the client.
// SaveGitHubToken creates a GitHub client with the given token and persists it.
// Unlike AuthenticateGitHub, this does NOT validate the token by fetching the user,
// so it returns immediately without any network call.
// Call ValidateGitHubToken() separately if you need the user profile.
func (e *Engine) SaveGitHubToken(token string) error {
	gh, err := github.NewCombinedClient(token)
	if err != nil {
		return fmt.Errorf("init github client: %w", err)
	}
	e.GitHub = gh
	e.Config.Set("github.token", token)
	return e.Config.Save()
}

// AuthenticateGitHub saves a GitHub token and validates it by fetching the user.
// The token is saved even if validation fails (with a 10s timeout).
// For instant dialog close without waiting for validation, use SaveGitHubToken instead.
func (e *Engine) AuthenticateGitHub(token string) error {
	gh, err := github.NewCombinedClient(token)
	if err != nil {
		return fmt.Errorf("init github client: %w", err)
	}

	// Save token immediately so it persists even if validation fails or times out.
	// This happens BEFORE the validation call so the state is durable.
	e.GitHub = gh
	e.Config.Set("github.token", token)
	if saveErr := e.Config.Save(); saveErr != nil {
		log.Printf("zgit: AuthenticateGitHub: save config: %v", saveErr)
	}

	// Validate with a short timeout — don't block the frontend login dialog for long.
	valCtx, valCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer valCancel()
	user, err := gh.GetAuthenticatedUser(valCtx)
	if err != nil {
		log.Printf("zgit: AuthenticateGitHub: user validation note (token already saved): %v", err)
	} else {
		e.Config.Set("github.user", user.Login)
		if saveErr := e.Config.Save(); saveErr != nil {
			log.Printf("zgit: AuthenticateGitHub: save user: %v", saveErr)
		}
	}
	return nil
}

// IsGitHubAuthenticated returns true if a GitHub client is available.
func (e *Engine) IsGitHubAuthenticated() bool {
	return e.GitHub != nil
}

// LogoutGitHub clears the GitHub token and client.
func (e *Engine) LogoutGitHub() error {
	e.GitHub = nil
	e.Config.Set("github.token", "")
	e.Config.Set("github.user", "")
	return e.Config.Save()
}

// NewAgent creates an AI agent wired to this engine's git adapter.
func (e *Engine) NewAgent(cfg ai.Config) (*ai.Agent, error) {
	return ai.NewAgent(cfg, e.Git)
}

// secondsToDuration converts seconds to time.Duration, 0 = no expiration.
func secondsToDuration(s int) time.Duration {
	if s <= 0 {
		return 0
	}
	return time.Duration(s) * time.Second
}

// guessCurrentRepo extracts owner/repo from the first origin/upstream remote.
func guessCurrentRepo(gitAdapter gitpkg.GitAdapter) (string, string, error) {
	remotes, err := gitAdapter.RemoteList(context.Background())
	if err != nil {
		return "", "", err
	}
	for _, r := range remotes {
		if r.Name == "origin" || r.Name == "upstream" {
			return guessOwnerFromRemoteURL(r.URL)
		}
	}
	if len(remotes) > 0 {
		return guessOwnerFromRemoteURL(remotes[0].URL)
	}
	return "", "", fmt.Errorf("no git remotes found")
}

// guessOwnerFromRemoteURL parses owner/repo from a GitHub remote URL.
func guessOwnerFromRemoteURL(url string) (string, string, error) {
	// Handle both HTTPS and SSH formats:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	url = stripSuffix(url, ".git")
	var hostPart string
	if contains(url, "github.com/") {
		hostPart = splitAfter(url, "github.com/")
	} else if contains(url, "github.com:") {
		hostPart = splitAfter(url, "github.com:")
	} else {
		return "", "", fmt.Errorf("not a GitHub remote: %s", url)
	}
	parts := split(hostPart, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("cannot parse owner/repo from %q", url)
}

// Helper functions to avoid importing strings everywhere
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stripSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func splitAfter(s, sep string) string {
	idx := indexOfStr(s, sep)
	if idx < 0 {
		return s
	}
	return s[idx+len(sep):]
}

func split(s, sep string) []string {
	var result []string
	for {
		idx := indexOfStr(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
