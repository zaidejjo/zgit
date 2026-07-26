package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/github"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// App is the main application struct for Wails.
// Its exported methods are automatically exposed to the frontend as Go bindings.
type App struct {
	engine      *core.Engine
	ctx         context.Context
	watcher     *fsnotify.Watcher
	watcherMu   sync.Mutex
	watcherDone chan struct{}
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
	a.restartFileWatcher()
}

// getGitDir uses git rev-parse to locate the actual .git directory.
// Handles worktrees (where .git is a file) and submodules correctly.
func (a *App) getGitDir() string {
	repoPath := a.engine.Git.Path()
	if repoPath == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 5e9)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("fsnotify: git rev-parse --git-dir failed: %v", err)
		return ""
	}
	dir := strings.TrimSpace(string(out))
	// If relative, make absolute
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoPath, dir)
	}
	return dir
}

// stopFileWatcher stops the current file watcher if running.
func (a *App) stopFileWatcher() {
	a.watcherMu.Lock()
	defer a.watcherMu.Unlock()
	if a.watcher != nil {
		a.watcher.Close()
		a.watcher = nil
	}
	if a.watcherDone != nil {
		close(a.watcherDone)
		a.watcherDone = nil
	}
}

// restartFileWatcher stops any existing watcher and starts a new one.
func (a *App) restartFileWatcher() {
	a.stopFileWatcher()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("fsnotify: create watcher: %v", err)
		return
	}

	repoPath := a.engine.Git.Path()
	if repoPath == "" {
		log.Println("fsnotify: no repo path, watcher disabled")
		watcher.Close()
		return
	}

	// Watch the working tree root
	if err := watcher.Add(repoPath); err != nil {
		log.Printf("fsnotify: watch repo %s: %v", repoPath, err)
		watcher.Close()
		return
	}

	// Watch the actual .git directory (handles worktrees)
	gitDir := a.getGitDir()
	if gitDir != "" && gitDir != repoPath {
		if err := watcher.Add(gitDir); err != nil {
			log.Printf("fsnotify: note: cannot watch %s: %v", gitDir, err)
		}
	}

	a.watcherMu.Lock()
	a.watcher = watcher
	done := make(chan struct{})
	a.watcherDone = done
	a.watcherMu.Unlock()

	// Debounce: coalesce rapid changes into one event
	var lastEvent time.Time
	const debounceInterval = 500 * time.Millisecond

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-done:
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if isIgnoredFile(event.Name) {
					continue
				}
				now := time.Now()
				if now.Sub(lastEvent) < debounceInterval {
					continue
				}
				lastEvent = now
				runtime.EventsEmit(a.ctx, "fs:status-changed", event.Name)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("fsnotify: error: %v", err)
			}
		}
	}()
}

// isIgnoredFile returns true for paths that should not trigger a refresh.
func isIgnoredFile(path string) bool {
	ignoredSuffixes := []string{".git/index.lock", ".git/HEAD.lock", ".git/refs/", ".git/objects/", ".git/logs/", "~", ".swp", ".swx"}
	for _, suffix := range ignoredSuffixes {
		if len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix {
			return true
		}
	}
	return false
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

// Commit creates a new commit with optional body.
func (a *App) Commit(message string, body string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	opts := git.CommitOptions{Message: message, Body: body}
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

// GitPush pushes the current branch to origin.
func (a *App) GitPush() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	opts := git.PushOptions{
		Remote:      "origin",
		SetUpstream: true,
	}
	return a.engine.Git.Push(ctx, opts)
}

// GitPushForce pushes the current branch with --force.
func (a *App) GitPushForce() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	opts := git.PushOptions{
		Remote:      "origin",
		SetUpstream: true,
		Force:       true,
	}
	return a.engine.Git.Push(ctx, opts)
}

// GitFetch fetches from the default remote.
func (a *App) GitFetch() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return a.engine.Git.Fetch(ctx, "", true)
}

// GitPull pulls latest changes (with optional rebase).
func (a *App) GitPull(rebase bool) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	opts := git.PullOptions{Rebase: rebase}
	return a.engine.Git.Pull(ctx, opts)
}

// StashList returns all stashed entries.
func (a *App) StashList() ([]*models.Stash, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.StashList(ctx)
}

// StashPush creates a new stash with an optional message.
func (a *App) StashPush(message string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.StashPush(ctx, git.StashOptions{}, message)
}

// StashPop pops (applies + drops) the stash at the given index.
func (a *App) StashPop(index int) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.StashPop(ctx, index)
}

// StashApply applies the stash at the given index without dropping it.
func (a *App) StashApply(index int) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.StashApply(ctx, index)
}

// StashDrop drops the stash at the given index.
func (a *App) StashDrop(index int) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.StashDrop(ctx, index)
}

// DiscardFile discards unstaged changes in the given file (git restore).
func (a *App) DiscardFile(file string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Restore(ctx, file)
}

// DiscardAllFiles discards all unstaged changes in the working tree.
func (a *App) DiscardAllFiles() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Restore(ctx, ".")
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
		return repo.Path
	}
	return ""
}

// GetRepoPath returns the filesystem path of the current repository.
func (a *App) GetRepoPath() string {
	return a.engine.Git.Path()
}

// ResolveGitRoot detects the nearest git repository root from the given path.
// Checks the path itself first, then searches parent directories (up to 5 levels).
// Returns the resolved git working tree root path.
func (a *App) ResolveGitRoot(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	// Check path and up to 5 parent directories for a .git entry
	dir := absPath
	for i := 0; i < 6; i++ {
		gitPath := filepath.Join(dir, ".git")
		if fi, statErr := os.Stat(gitPath); statErr == nil {
			// .git exists — this is a git repo
			_ = fi // .git can be dir (normal) or file (worktree)
			return dir, nil
		}
		// Try git rev-parse as fallback (handles worktrees, submodules)
		ctx, cancel := context.WithTimeout(a.getContext(), 3e9)
		cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--show-toplevel")
		out, runErr := cmd.Output()
		cancel()
		if runErr == nil {
			root := strings.TrimSpace(string(out))
			return root, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return "", fmt.Errorf("no git repository found in or above %s", absPath)
}

// OpenRepo opens a new repository at the given path.
// It resolves the git root, opens the repo, saves to recent list,
// restarts the file watcher, and emits a repo:switched event to the frontend.
func (a *App) OpenRepo(path string) error {
	resolved, err := a.ResolveGitRoot(path)
	if err != nil {
		return err
	}
	if err := a.engine.OpenRepo(resolved); err != nil {
		return fmt.Errorf("open repo: %w", err)
	}
	// Save to recent repos
	a.engine.Config.AddRecentRepo(resolved)
	if err := a.engine.Config.Save(); err != nil {
		log.Printf("warning: save config after AddRecentRepo: %v", err)
	}
	// Restart file watcher for the new repo
	a.restartFileWatcher()
	// Notify frontend
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "repo:switched", resolved)
	}
	return nil
}

// ListRecentRepos returns recently opened repository paths.
func (a *App) ListRecentRepos() []string {
	return a.engine.Config.GetRecentRepos()
}

// PickDirectory opens a native directory picker dialog and returns the selected path.
// Returns empty string if the user cancelled.
func (a *App) PickDirectory() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app context not ready")
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Git Repository",
	})
	if err != nil {
		return "", fmt.Errorf("directory picker: %w", err)
	}
	return dir, nil
}

// GetRepoName returns a human-readable name for the current repo.
func (a *App) GetRepoName() string {
	path := a.engine.Git.Path()
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

// --- Workflow / CI operations ---

// ListWorkflowRuns returns recent workflow runs.
func (a *App) ListWorkflowRuns() ([]*models.WorkflowRun, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return []*models.WorkflowRun{}, nil
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return []*models.WorkflowRun{}, nil
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	runs, err := gh.ListWorkflowRuns(ctx, owner, repo, github.RunsFilter{
		Limit: 30,
	})
	if err != nil {
		return []*models.WorkflowRun{}, nil
	}
	return runs, nil
}

// ReRunWorkflow re-runs a workflow run.
func (a *App) ReRunWorkflow(runID int64) error {
	gh, err := a.getGitHubClient()
	if err != nil {
		return err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.ReRunWorkflow(ctx, owner, repo, runID)
}

// CancelWorkflowRun cancels a workflow run.
func (a *App) CancelWorkflowRun(runID int64) error {
	gh, err := a.getGitHubClient()
	if err != nil {
		return err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.CancelWorkflowRun(ctx, owner, repo, runID)
}

// ListWorkflowJobs returns jobs for a workflow run.
func (a *App) ListWorkflowJobs(runID int64) ([]*models.Job, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return []*models.Job{}, nil
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return []*models.Job{}, nil
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.ListWorkflowJobs(ctx, owner, repo, runID)
}

// GetWorkflowJobLogs returns the logs for a specific job.
func (a *App) GetWorkflowJobLogs(jobID int64) (string, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return "", err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return gh.GetWorkflowJobLogs(ctx, owner, repo, jobID)
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
