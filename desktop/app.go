package main

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// App is the main application struct for Wails.
// Its exported methods are automatically exposed to the frontend as Go bindings.
type App struct {
	engine *core.Engine
	ctx    context.Context
}

// NewApp creates a new App with the given engine.
func NewApp(engine *core.Engine) *App {
	return &App{engine: engine}
}

// Run starts the Wails application.
func (a *App) Run(assets embed.FS) error {
	return wails.Run(&options.App{
		Title:     "zgit",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		Assets:    assets,
		OnStartup: a.startup,
		Bind: []any{
			a,
		},
		Linux: &linux.Options{
			ProgramName: "zgit",
		},
	})
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// getContext returns the Wails context if available, falling back to Background().
// This prevents "cannot create context from nil parent" errors when the frontend
// makes API calls before startup() has been invoked.
func (a *App) getContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// --- Git operations ---

// GetStatus returns the current working tree status.
func (a *App) GetStatus() (*models.Status, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Status(ctx)
}

// GetLog returns commit history.
func (a *App) GetLog(count int) ([]*models.Commit, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Log(ctx, git.LogOptions{Count: count})
}

// GetBranches returns all local branches.
func (a *App) GetBranches() ([]*models.Branch, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Branches(ctx)
}

// GetDiff returns the diff for a specific file or the entire working tree.
func (a *App) GetDiff(pathspec string) (*models.Diff, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	opts := git.DiffOptions{
		Unified:  true,
		Pathspec: pathspec,
	}
	if pathspec == "" {
		opts.Unified = true
	}
	return a.engine.Git.Diff(ctx, opts)
}

// GetFileDiff returns the diff for a single file with full patch text.
func (a *App) GetFileDiff(pathspec string) (*models.Diff, error) {
	return a.GetDiff(pathspec)
}

// StageFile stages a file.
func (a *App) StageFile(file string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Add(ctx, git.AddOptions{}, file)
}

// UnstageFile unstages a file.
func (a *App) UnstageFile(file string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Reset(ctx, file)
}

// StageAll stages all changes.
func (a *App) StageAll() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Add(ctx, git.AddOptions{All: true})
}

// UnstageAll unstages all changes.
func (a *App) UnstageAll() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Reset(ctx)
}

// Commit creates a new commit.
func (a *App) Commit(message string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	opts := git.CommitOptions{Message: message}
	if err := a.engine.Git.Commit(ctx, opts); err != nil {
		return "", err
	}
	// Return the latest commit hash
	commits, err := a.engine.Git.Log(ctx, git.LogOptions{Count: 1})
	if err != nil || len(commits) == 0 {
		return "", nil
	}
	return commits[0].Hash, nil
}

// CheckoutBranch checks out a branch.
func (a *App) CheckoutBranch(name string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Checkout(ctx, name)
}

// CreateBranch creates and checks out a new branch.
func (a *App) CreateBranch(name string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.CreateBranchAndCheckout(ctx, name)
}

// DeleteBranch deletes a branch.
func (a *App) DeleteBranch(name string, force bool) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.BranchDelete(ctx, name, force)
}

// CurrentBranch returns the current branch name.
func (a *App) CurrentBranch() (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.CurrentBranch(ctx)
}

// --- GitHub operations ---

// IsGitHubAuthenticated returns whether a GitHub token is configured.
func (a *App) IsGitHubAuthenticated() bool {
	return a.engine.IsGitHubAuthenticated()
}

// AuthenticateGitHub sets a GitHub PAT and initializes the client.
func (a *App) AuthenticateGitHub(token string) error {
	return a.engine.AuthenticateGitHub(token)
}

// StartDeviceFlow initiates GitHub OAuth Device Flow.
// Returns the device code, user code, verification URI, and polling interval.
func (a *App) StartDeviceFlow() (*models.DeviceFlowCode, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()

	clientID := a.engine.Config.GetString("github.device_flow_client_id")
	if clientID == "" {
		clientID = github.DefaultDeviceFlowClientID
	}

	token, err := github.StartDeviceFlow(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return &models.DeviceFlowCode{
		DeviceCode:      token.DeviceCode,
		UserCode:        token.UserCode,
		VerificationURI: token.VerificationURI,
		Interval:        token.Interval,
	}, nil
}

// PollDeviceFlow polls GitHub for device flow authorization.
// Returns the access token when authorized, or empty string if still pending.
// The caller should retry after the interval from StartDeviceFlow.
func (a *App) PollDeviceFlow(deviceCode string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()

	clientID := a.engine.Config.GetString("github.device_flow_client_id")
	if clientID == "" {
		clientID = github.DefaultDeviceFlowClientID
	}

	token, err := github.PollDeviceFlow(ctx, clientID, deviceCode)
	if err != nil {
		// If it's an authorization_pending error, return empty string — caller retries
		errStr := err.Error()
		if strings.Contains(errStr, "authorization_pending") {
			return "", nil
		}
		return "", err
	}
	return token, nil
}

// GetGitHubUser returns the authenticated GitHub user.
func (a *App) GetGitHubUser() (*models.User, error) {
	if !a.engine.IsGitHubAuthenticated() {
		return nil, fmt.Errorf("GitHub not authenticated")
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.GitHub.GetAuthenticatedUser(ctx)
}

// getGitHubClient returns the GitHub client or an error if not authenticated.
func (a *App) getGitHubClient() (github.GitHubClient, error) {
	if !a.engine.IsGitHubAuthenticated() {
		return nil, fmt.Errorf("GitHub not authenticated — set a token in Settings")
	}
	return a.engine.GitHub, nil
}

// GetPullRequests lists open pull requests.
func (a *App) GetPullRequests() ([]*models.PullRequestSummary, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return []*models.PullRequestSummary{}, nil // empty, not error
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return []*models.PullRequestSummary{}, nil
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	prs, err := gh.ListPullRequests(ctx, owner, repo, github.PRFilter{
		State: "open",
		Sort:  "updated",
		Limit: 50,
	})
	if err != nil {
		return []*models.PullRequestSummary{}, nil
	}
	if prs == nil {
		return []*models.PullRequestSummary{}, nil
	}
	return prs, nil
}

// GetIssues lists open issues.
func (a *App) GetIssues() ([]*models.Issue, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return []*models.Issue{}, nil
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return []*models.Issue{}, nil
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	issues, err := gh.ListIssues(ctx, owner, repo, github.IssuesFilter{
		State: "open",
		Sort:  "updated",
		Limit: 50,
	})
	if err != nil {
		return []*models.Issue{}, nil
	}
	if issues == nil {
		return []*models.Issue{}, nil
	}
	return issues, nil
}

// GetRepository returns repo info from GitHub.
func (a *App) GetRepository() (*models.Repo, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return gh.GetRepository(ctx, owner, repo)
}

// GetCurrentRepoPath returns the current repo path.
func (a *App) GetCurrentRepoPath() string {
	repo := a.engine.CurrentRepo()
	if repo != nil {
		return repo.FullName
	}
	return ""
}

// guessRepo extracts owner/repo from git remotes.
func (a *App) guessRepo() (string, string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 5e9)
	defer cancel()
	remotes, err := a.engine.Git.RemoteList(ctx)
	if err != nil {
		return "", "", fmt.Errorf("list remotes: %w", err)
	}
	for _, r := range remotes {
		if r.Name == "origin" || r.Name == "upstream" {
			return parseRemoteURL(r.URL)
		}
	}
	if len(remotes) > 0 {
		return parseRemoteURL(remotes[0].URL)
	}
	return "", "", fmt.Errorf("no git remotes found")
}

// parseRemoteURL extracts owner/repo from a GitHub remote URL.
func parseRemoteURL(url string) (string, string, error) {
	// Strip .git suffix
	if len(url) > 4 && url[len(url)-4:] == ".git" {
		url = url[:len(url)-4]
	}

	var path string
	for _, sep := range []string{"github.com/", "github.com:"} {
		for i := 0; i <= len(url)-len(sep); i++ {
			if url[i:i+len(sep)] == sep {
				path = url[i+len(sep):]
				break
			}
		}
		if path != "" {
			break
		}
	}
	if path == "" {
		return "", "", fmt.Errorf("not a GitHub remote: %s", url)
	}

	parts := splitPath(path, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("cannot parse owner/repo from %q", url)
}

func splitPath(s, sep string) []string {
	var result []string
	for {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
