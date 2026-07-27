package main

import (
	"context"
	"embed"
	"encoding/json"
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
	"github.com/zaidejjo/zgit/pkg/core/ai"
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
	agent       *ai.Agent
	agentMu     sync.Mutex

	sessionManager *ai.SessionManager
	askCancel      context.CancelFunc
	askCancelMu    sync.Mutex
}

// NewApp creates a new App with the given engine.
func NewApp(engine *core.Engine) *App {
	return &App{
		engine:         engine,
		sessionManager: ai.NewSessionManager(),
	}
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

// GetGraphLog returns commit history with parent hashes, in topo-order for graph rendering.
func (a *App) GetGraphLog(count int) ([]*models.Commit, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Log(ctx, git.LogOptions{Count: count, Graph: true})
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

// StagePatch applies a patch hunk to the index for line-level staging.
// patch is a unified-diff hunk text.
func (a *App) StagePatch(patch string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.ApplyPatch(ctx, patch, true)
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

// GetRemotes returns the list of git remotes.
func (a *App) GetRemotes() ([]*models.Remote, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.RemoteList(ctx)
}

// AddRemote adds a new remote.
func (a *App) AddRemote(name, url string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.RemoteAdd(ctx, name, url)
}

// RemoveRemote removes a remote.
func (a *App) RemoveRemote(name string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.RemoteRemove(ctx, name)
}

// GetAheadCommits returns commits ahead of the remote tracking branch.
// Uses `git log origin/<branch>..HEAD` to list unpushed commits.
func (a *App) GetAheadCommits() ([]*models.Commit, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	branch, err := a.engine.Git.CurrentBranch(ctx)
	if err != nil {
		return nil, err
	}
	opts := git.LogOptions{
		Count:  100,
		Branch: fmt.Sprintf("origin/%s..HEAD", branch),
	}
	commits, err := a.engine.Git.Log(ctx, opts)
	if err != nil {
		return []*models.Commit{}, nil
	}
	return commits, nil
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

// GitRenameBranch renames a local branch.
func (a *App) GitRenameBranch(oldName, newName string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.BranchRename(ctx, oldName, newName)
}

// CurrentBranch returns the current branch name.
func (a *App) CurrentBranch() (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.CurrentBranch(ctx)
}

// GitMerge merges the given branch into the current branch.
func (a *App) GitMerge(branch string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return a.engine.Git.Merge(ctx, branch)
}

// RebaseSequence executes an interactive rebase sequence.
func (a *App) RebaseSequence(onto string, commitsJSON string) (*models.RebaseResult, error) {
	var opts models.RebaseSequenceOptions
	opts.Onto = onto
	if err := json.Unmarshal([]byte(commitsJSON), &opts.Commits); err != nil {
		return nil, fmt.Errorf("parse rebase commits: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return a.engine.Git.RebaseSequence(ctx, opts)
}

// CherryPick applies a single commit onto the current HEAD.
func (a *App) CherryPick(sha string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return a.engine.Git.CherryPick(ctx, sha)
}

// RevertCommit reverts a single commit.
func (a *App) RevertCommit(sha string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return a.engine.Git.Revert(ctx, sha)
}

// ResetCommit moves HEAD to a commit with the given mode (soft, mixed, hard).
func (a *App) ResetCommit(sha, mode string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return a.engine.Git.ResetCommit(ctx, sha, mode)
}

// TagList returns all tags sorted by creation date (newest first).
func (a *App) TagList() ([]string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.TagList(ctx)
}

// TagCreate creates a tag. If message is non-empty, creates an annotated tag.
func (a *App) TagCreate(name, target, message string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.TagCreate(ctx, name, target, message)
}

// TagDelete deletes a tag.
func (a *App) TagDelete(name string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.TagDelete(ctx, name)
}

// ConfigGet gets a git config value.
func (a *App) ConfigGet(key string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 5e9)
	defer cancel()
	return a.engine.Git.ConfigGet(ctx, key)
}

// ConfigSet sets a git config value.
func (a *App) ConfigSet(key, value string, global bool) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 5e9)
	defer cancel()
	return a.engine.Git.ConfigSet(ctx, key, value, global)
}

// CheckoutOurs resolves a conflicted file using the --ours version.
func (a *App) CheckoutOurs(file string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.CheckoutOurs(ctx, file)
}

// CheckoutTheirs resolves a conflicted file using the --theirs version.
func (a *App) CheckoutTheirs(file string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.CheckoutTheirs(ctx, file)
}

// --- AI Commit Message Generator ---

// GetAIConfig returns the current AI configuration.
func (a *App) GetAIConfig() (models.AIConfig, error) {
	cfg := a.engine.Config
	return models.AIConfig{
		Provider: cfg.GetString("ai.provider"),
		APIKey:   cfg.GetString("ai.api_key"),
		Model:    cfg.GetString("ai.model"),
		Endpoint: cfg.GetString("ai.endpoint"),
	}, nil
}

// SetAIConfig saves the AI provider configuration.
func (a *App) SetAIConfig(provider, apiKey, model, endpoint string) error {
	cfg := a.engine.Config
	cfg.Set("ai.provider", provider)
	cfg.Set("ai.api_key", apiKey)
	cfg.Set("ai.model", model)
	cfg.Set("ai.endpoint", endpoint)
	return cfg.Save()
}

// GenerateCommitMessage reads the staged diff and returns an AI-generated conventional commit message.
func (a *App) GenerateCommitMessage() (string, error) {
	// 1. Get staged diff
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()

	diffOpts := git.DiffOptions{Cached: true, Unified: true}
	diff, err := a.engine.Git.Diff(ctx, diffOpts)
	if err != nil {
		return "", fmt.Errorf("get staged diff: %w", err)
	}

	// Build diff text from files
	var diffText string
	for _, f := range diff.Files {
		if f.UnifiedDiff != "" {
			diffText += f.UnifiedDiff + "\n"
		}
	}

	if strings.TrimSpace(diffText) == "" {
		return "", fmt.Errorf("no staged changes — stage files before generating a commit message")
	}

	// 2. Get AI config
	aiCfg := ai.Config{
		Provider: ai.ProviderKind(a.engine.Config.GetString("ai.provider")),
		APIKey:   a.engine.Config.GetString("ai.api_key"),
		Model:    a.engine.Config.GetString("ai.model"),
		Endpoint: a.engine.Config.GetString("ai.endpoint"),
	}

	if aiCfg.Provider == "" {
		return "", fmt.Errorf("AI provider not configured — go to Settings to set up AI commit message generation")
	}
	if aiCfg.APIKey == "" {
		return "", fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	// 3. Generate
	generator, err := ai.NewGenerator(aiCfg)
	if err != nil {
		return "", err
	}

	msg, err := generator.GenerateCommitMessage(ctx, diffText, aiCfg)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	return msg, nil
}

// --- Agentic AI Assistant ---

// AgentStart creates a new agent session with the configured provider.
func (a *App) AgentStart() error {
	a.agentMu.Lock()
	defer a.agentMu.Unlock()

	aiCfg := ai.Config{
		Provider: ai.ProviderKind(a.engine.Config.GetString("ai.provider")),
		APIKey:   a.engine.Config.GetString("ai.api_key"),
		Model:    a.engine.Config.GetString("ai.model"),
		Endpoint: a.engine.Config.GetString("ai.endpoint"),
	}

	maxTurns := a.engine.Config.GetInt("ai.max_turns")
	if maxTurns > 0 {
		aiCfg.MaxTurns = maxTurns
	}
	aiCfg.AutoMode = a.engine.Config.GetBool("ai.auto_mode")

	if aiCfg.Provider == "" {
		return fmt.Errorf("AI provider not configured — go to Settings to set up AI")
	}
	if aiCfg.APIKey == "" {
		return fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	agent, err := a.engine.NewAgent(aiCfg)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	// Register additional tools that need full engine access
	agent.RegisterTool(ai.NewAutoResolveConflictTool(a.engine.Git))
	agent.RegisterTool(ai.NewGeneratePRReviewTool(a.engine.Git))

	a.agent = agent
	return nil
}

// AgentChat sends a message to the AI agent and returns its response.
func (a *App) AgentChat(message string) (*ai.AgentResponse, error) {
	a.agentMu.Lock()
	agent := a.agent
	a.agentMu.Unlock()

	if agent == nil {
		return nil, fmt.Errorf("no active agent session — call AgentStart first")
	}

	ctx, cancel := context.WithTimeout(a.getContext(), 180e9)
	defer cancel()

	return agent.Chat(ctx, message)
}

// AgentApproveProposal executes an approved proposal.
func (a *App) AgentApproveProposal(proposalID string) (*ai.ProposalResult, error) {
	a.agentMu.Lock()
	agent := a.agent
	a.agentMu.Unlock()

	if agent == nil {
		return nil, fmt.Errorf("no active agent session")
	}

	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()

	return agent.ApproveProposal(ctx, proposalID)
}

// AgentRejectProposal rejects a proposal with optional feedback.
func (a *App) AgentRejectProposal(proposalID string, feedback string) error {
	a.agentMu.Lock()
	agent := a.agent
	a.agentMu.Unlock()

	if agent == nil {
		return fmt.Errorf("no active agent session")
	}

	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()

	return agent.RejectProposal(ctx, proposalID, feedback)
}

// AgentReset clears the current agent session.
func (a *App) AgentReset() error {
	a.agentMu.Lock()
	defer a.agentMu.Unlock()

	if a.agent != nil {
		a.agent.Reset()
		a.agent = nil
	}
	return nil
}

// AgentGetProposals returns all pending proposals.
func (a *App) AgentGetProposals() ([]ai.AgentActionProposal, error) {
	a.agentMu.Lock()
	agent := a.agent
	a.agentMu.Unlock()

	if agent == nil {
		return nil, nil
	}
	return agent.GetPendingProposals(), nil
}

// AgentGetHistory returns the raw conversation history.
func (a *App) AgentGetHistory() ([]ai.Message, error) {
	a.agentMu.Lock()
	agent := a.agent
	a.agentMu.Unlock()

	if agent == nil {
		return nil, nil
	}
	return agent.History(), nil
}

// --- Ask Mode (tool-less Q&A) ---

// getAskConfig reads AI config from viper and returns a Config for Ask mode.
func (a *App) getAskConfig() ai.Config {
	return ai.Config{
		Provider: ai.ProviderKind(a.engine.Config.GetString("ai.provider")),
		APIKey:   a.engine.Config.GetString("ai.api_key"),
		Model:    a.engine.Config.GetString("ai.model"),
		Endpoint: a.engine.Config.GetString("ai.endpoint"),
	}
}

// AskChat sends a tool-less Q&A message and returns the full response.
// Uses the active Ask session, or creates one if none exists.
func (a *App) AskChat(message string) (string, error) {
	aiCfg := a.getAskConfig()
	if aiCfg.Provider == "" {
		return "", fmt.Errorf("AI provider not configured — go to Settings to set up AI")
	}
	if aiCfg.APIKey == "" {
		return "", fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	provider, err := ai.NewAskProvider(aiCfg)
	if err != nil {
		return "", fmt.Errorf("create ask provider: %w", err)
	}

	// Ensure active Ask session
	session := a.sessionManager.Active()
	if session == nil || session.Mode != "ask" {
		session, err = a.sessionManager.Create("Ask Session", "ask")
		if err != nil {
			return "", fmt.Errorf("create session: %w", err)
		}
	}

	// Add user message to session
	if err := a.sessionManager.AddMessage(session.ID, ai.Message{Role: "user", Content: message}); err != nil {
		return "", fmt.Errorf("add message: %w", err)
	}

	// Get full message history for context
	messages, err := a.sessionManager.GetMessages(session.ID)
	if err != nil {
		return "", fmt.Errorf("get messages: %w", err)
	}

	// Send to provider (blocking)
	resp, err := provider.Ask(a.getContext(), messages)
	if err != nil {
		return "", fmt.Errorf("ask failed: %w", err)
	}

	// Add assistant response to session
	if err := a.sessionManager.AddMessage(session.ID, resp); err != nil {
		return "", fmt.Errorf("add response: %w", err)
	}

	return resp.Content, nil
}

// AskChatStream starts a streaming Q&A call. Tokens are emitted via Wails runtime
// events ("ai:token"). Completion emits "ai:done" with full content. Errors emit "ai:error".
func (a *App) AskChatStream(message string) error {
	aiCfg := a.getAskConfig()
	if aiCfg.Provider == "" {
		runtime.EventsEmit(a.ctx, "ai:error", "AI provider not configured — go to Settings to set up AI")
		return nil
	}
	if aiCfg.APIKey == "" {
		runtime.EventsEmit(a.ctx, "ai:error", "API key not configured — go to Settings to set your API key")
		return nil
	}

	provider, err := ai.NewAskProvider(aiCfg)
	if err != nil {
		runtime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("create ask provider: %s", err.Error()))
		return nil
	}

	// Ensure active Ask session
	session := a.sessionManager.Active()
	if session == nil || session.Mode != "ask" {
		session, err = a.sessionManager.Create("Ask Session", "ask")
		if err != nil {
			runtime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("create session: %s", err.Error()))
			return nil
		}
	}

	// Add user message
	if err := a.sessionManager.AddMessage(session.ID, ai.Message{Role: "user", Content: message}); err != nil {
		runtime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("add message: %s", err.Error()))
		return nil
	}

	messages, err := a.sessionManager.GetMessages(session.ID)
	if err != nil {
		runtime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("get messages: %s", err.Error()))
		return nil
	}

	// Create cancellable context for streaming
	ctx, cancel := context.WithCancel(a.getContext())
	a.askCancelMu.Lock()
	if a.askCancel != nil {
		a.askCancel()
	}
	a.askCancel = cancel
	a.askCancelMu.Unlock()

	// Stream in background
	go func() {
		defer cancel()

		resp, err := provider.AskStream(ctx, messages, func(token string) {
			runtime.EventsEmit(a.ctx, "ai:token", token)
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "ai:error", err.Error())
			return
		}

		// Add assistant response to session
		if err := a.sessionManager.AddMessage(session.ID, resp); err != nil {
			runtime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("save response: %s", err.Error()))
			return
		}

		runtime.EventsEmit(a.ctx, "ai:done", resp.Content)
	}()

	return nil
}

// AskChatCancel cancels any in-flight streaming Ask call.
func (a *App) AskChatCancel() error {
	a.askCancelMu.Lock()
	defer a.askCancelMu.Unlock()
	if a.askCancel != nil {
		a.askCancel()
		a.askCancel = nil
	}
	return nil
}

// AgentCancel cancels any in-flight agent Chat call.
func (a *App) AgentCancel() error {
	a.agentMu.Lock()
	defer a.agentMu.Unlock()
	if a.agent != nil {
		a.agent.Reset()
		a.agent = nil
	}
	return nil
}

// --- Session Management ---

// SessionCreate creates a new session with the given name and mode ("ask" or "agent").
func (a *App) SessionCreate(name, mode string) (*ai.SessionSummary, error) {
	s, err := a.sessionManager.Create(name, mode)
	if err != nil {
		return nil, err
	}
	return &ai.SessionSummary{
		ID:           s.ID,
		Name:         s.Name,
		Mode:         s.Mode,
		MessageCount: len(s.Messages),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}, nil
}

// SessionList returns summaries of all sessions.
func (a *App) SessionList() ([]ai.SessionSummary, error) {
	return a.sessionManager.List(), nil
}

// SessionRename renames a session.
func (a *App) SessionRename(id, name string) error {
	return a.sessionManager.Rename(id, name)
}

// SessionDelete deletes a session.
func (a *App) SessionDelete(id string) error {
	return a.sessionManager.Delete(id)
}

// SessionSwitch switches the active session.
func (a *App) SessionSwitch(id string) (*ai.SessionSummary, error) {
	s, err := a.sessionManager.Switch(id)
	if err != nil {
		return nil, err
	}
	return &ai.SessionSummary{
		ID:           s.ID,
		Name:         s.Name,
		Mode:         s.Mode,
		MessageCount: len(s.Messages),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}, nil
}

// SessionGetMessages returns all messages for a session.
func (a *App) SessionGetMessages(id string) ([]ai.Message, error) {
	return a.sessionManager.GetMessages(id)
}

// SessionClearMessages clears a session's messages (keeps system prompt).
func (a *App) SessionClearMessages(id string) error {
	return a.sessionManager.ClearMessages(id)
}

// SessionActiveID returns the ID of the currently active session.
func (a *App) SessionActiveID() (string, error) {
	id := a.sessionManager.ActiveID()
	if id == "" {
		return "", fmt.Errorf("no active session")
	}
	return id, nil
}

// --- 3-Way Merge Editor ---

// GetConflictFiles returns all files with unresolved merge conflicts.
func (a *App) GetConflictFiles() ([]models.ConflictFile, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.ConflictFiles(ctx)
}

// GetMergeConflictDetail returns 3-way conflict detail for a specific file.
func (a *App) GetMergeConflictDetail(file string) (*models.MergeConflictDetail, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return a.engine.Git.GetMergeConflictDetail(ctx, file)
}

// StageResolvedFile writes resolved content and stages it.
func (a *App) StageResolvedFile(file, content string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.StageResolvedFile(ctx, file, content)
}

// --- Reflog / Undo ---

// GetReflog returns the last N reflog entries.
func (a *App) GetReflog(count int) ([]models.ReflogEntry, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.Reflog(ctx, count)
}

// UndoLastAction undoes the most recent reflog entry and returns a description.
func (a *App) UndoLastAction() (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return a.engine.Git.UndoLastAction(ctx)
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

// GetIssueDetail fetches a single issue with its comments.
func (a *App) GetIssueDetail(number int) (*models.Issue, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.GetIssue(ctx, owner, repo, number)
}

// CreateIssue opens a new issue on GitHub.
func (a *App) CreateIssue(title, body string) (*models.Issue, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.CreateIssue(ctx, owner, repo, github.IssueRequest{
		Title: title,
		Body:  body,
	})
}

// CloseIssue closes an issue on GitHub.
func (a *App) CloseIssue(number int) error {
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
	return gh.CloseIssue(ctx, owner, repo, number)
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

// GetPullRequestDetail fetches full PR detail with commits, reviews, checks.
func (a *App) GetPullRequestDetail(number int) (*models.PullRequestDetail, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.GetPullRequest(ctx, owner, repo, number)
}

// CreatePullRequest creates a new pull request on GitHub.
func (a *App) CreatePullRequest(title, body, head, base string, draft bool) (*models.PullRequestSummary, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.CreatePullRequest(ctx, owner, repo, github.PRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
		Draft: draft,
	})
}

// MergePullRequest merges a pull request with the given method.
// method: "merge", "squash", "rebase"
func (a *App) MergePullRequest(number int, method string) error {
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
	return gh.MergePullRequest(ctx, owner, repo, number, method)
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
