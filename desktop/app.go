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
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/ai"
	"github.com/zaidejjo/zgit/pkg/core/config"
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

	sessionManager   *ai.SessionManager
	askCancel        context.CancelFunc
	askCancelMu      sync.Mutex
	devicePollCancel context.CancelFunc
	devicePollMu     sync.Mutex
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

// chatsFilePath returns the legacy AI sessions JSON file, used once to
// migrate into SQLite. New history lives in the store (see db ai_* tables).
func (a *App) chatsFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "zgit", "chats.json")
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Chat history persists in SQLite with per-message write-through,
	// so no full-file rewrite happens on every turn.
	a.sessionManager.AttachStore(a.engine.Store)
	if migrated, err := a.sessionManager.MigrateFileToStore(a.chatsFilePath()); err != nil {
		log.Printf("migrate chats: %v", err)
	} else if migrated {
		log.Printf("migrated chats.json into sqlite")
	}
	if err := a.sessionManager.LoadFromStore(); err != nil {
		log.Printf("load chats: %v", err)
	}
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
				wailsRuntime.EventsEmit(a.ctx, "fs:status-changed", event.Name)
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
// It skips git internals, editor temp files, and common dependency/build
// directories so changelogs don't churn on large node_modules/target/etc.
func isIgnoredFile(path string) bool {
	ignoredSuffixes := []string{".git/index.lock", ".git/HEAD.lock", ".git/refs/", ".git/objects/", ".git/logs/", "~", ".swp", ".swx"}
	for _, suffix := range ignoredSuffixes {
		if len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix {
			return true
		}
	}
	// Skip common dependency/build/output directories anywhere in the path.
	ignoredDirs := []string{"node_modules", ".git", "target", "build", "dist", ".next", "vendor", ".venv", "__pycache__"}
	parts := splitPath(path, string(filepath.Separator))
	for _, p := range parts {
		for _, d := range ignoredDirs {
			if p == d {
				return true
			}
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

// GetBlame returns per-line commit attribution for a file.
func (a *App) GetBlame(path, ref, lines string) ([]*models.BlameLine, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	result, err := a.engine.Git.Blame(ctx, path, models.BlameOptions{Ref: ref, Lines: lines})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []*models.BlameLine{}, nil
	}
	return result, nil
}

// GetFileHistory returns commit history for a file, optionally following
// renames (--follow) or filtering by pickaxe (-S/-G).
func (a *App) GetFileHistory(path, branch string, follow bool, pickaxe, pickaxeRe string, count int) ([]*models.Commit, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	result, err := a.engine.Git.Log(ctx, git.LogOptions{
		Count:    count,
		Branch:   branch,
		File:     path,
		Follow:   follow,
		Pickaxe:  pickaxe,
		PickaxeR: pickaxeRe,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []*models.Commit{}, nil
	}
	return result, nil
}

// GetRepoFiles returns all tracked file paths (for completion pickers).
func (a *App) GetRepoFiles() ([]string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	files, err := a.engine.Git.ListFiles(ctx)
	if err != nil {
		return []string{}, nil
	}
	if files == nil {
		return []string{}, nil
	}
	return files, nil
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

// RenameRemote renames a remote from oldName to newName.
func (a *App) RenameRemote(oldName, newName string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.RemoteRename(ctx, oldName, newName)
}

// SetRemoteURL updates the URL of a remote.
func (a *App) SetRemoteURL(name, url string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.RemoteSetURL(ctx, name, url)
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

// --- Worktrees ---

// GetWorktrees returns all linked worktrees (main first).
func (a *App) GetWorktrees() ([]*models.Worktree, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	list, err := a.engine.Git.WorktreeList(ctx)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return []*models.Worktree{}, nil
	}
	return list, nil
}

// AddWorktree creates a new worktree at path checking out ref.
// When createBranch is true, ref is treated as a new branch name.
func (a *App) AddWorktree(path, ref string, createBranch, force, detach bool) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return a.engine.Git.WorktreeAdd(ctx, path, ref, models.WorktreeAddOptions{
		CreateBranch: createBranch,
		Force:        force,
		Detach:       detach,
	})
}

// RemoveWorktree removes the worktree at path.
func (a *App) RemoveWorktree(path string, force bool) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return a.engine.Git.WorktreeRemove(ctx, path, force)
}

// LockWorktree locks a worktree with an optional reason.
func (a *App) LockWorktree(path, reason string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.WorktreeLock(ctx, path, reason)
}

// UnlockWorktree unlocks a worktree.
func (a *App) UnlockWorktree(path string) error {
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.Git.WorktreeUnlock(ctx, path)
}

// PruneWorktrees prunes metadata for worktrees deleted on disk.
func (a *App) PruneWorktrees() error {
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return a.engine.Git.WorktreePrune(ctx)
}

// --- Repository dashboard ---

// GetRepoDashboard returns status overviews for favorite + recent repos.
// Snapshots run in parallel; unreadable paths come back with Error set.
func (a *App) GetRepoDashboard(force bool) ([]*models.RepoOverview, error) {
	paths := a.engine.DashboardRepos()
	out := make([]*models.RepoOverview, len(paths))
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			out[i] = a.engine.RepoOverview(p, force)
		}(i, p)
	}
	wg.Wait()
	return out, nil
}

// ToggleFavoriteRepo pins or unpins a repo. Returns true when now pinned.
func (a *App) ToggleFavoriteRepo(path string) (bool, error) {
	return a.engine.ToggleFavoriteRepo(path)
}

// ScanForRepos finds git repositories under root (4 levels deep).
func (a *App) ScanForRepos(root string) ([]models.ScannedRepo, error) {
	return a.engine.ScanForRepos(root, 4)
}

// TrackRepos records repos in the app. Returns how many were new.
func (a *App) TrackRepos(paths []string) (int, error) {
	return a.engine.TrackRepos(paths), nil
}

// FetchAllRepos fetches the default remote of every dashboard repo.
// Returns the number of repos fetched; the first failure aborts.
func (a *App) FetchAllRepos() (int, error) {
	count := 0
	for _, p := range a.engine.DashboardRepos() {
		if err := a.engine.FetchRepo(p); err != nil {
			return count, fmt.Errorf("fetch %s: %w", p, err)
		}
		count++
	}
	return count, nil
}

// PruneDeadRepos drops recent entries whose paths no longer exist.
// Returns the number removed.
func (a *App) PruneDeadRepos() (int, error) {
	return a.engine.PruneDeadRepos()
}

// RemoveRepo forgets a repo in the app (files on disk are untouched).
// If it was the open repo, the caller should open another one after.
func (a *App) RemoveRepo(path string) error {
	return a.engine.RemoveRepo(path)
}

// --- Projects ---

// GetProjects returns all projects with member paths.
func (a *App) GetProjects() ([]models.Project, error) {
	projects := a.engine.GetProjects()
	if projects == nil {
		return []models.Project{}, nil
	}
	return projects, nil
}

// CreateProject creates a project and returns its id.
func (a *App) CreateProject(name, color string) (string, error) {
	return a.engine.CreateProject(name, color)
}

// RenameProject renames a project.
func (a *App) RenameProject(id, name string) error {
	return a.engine.RenameProject(id, name)
}

// SetProjectColor changes a project's accent color.
func (a *App) SetProjectColor(id, color string) error {
	return a.engine.SetProjectColor(id, color)
}

// DeleteProject removes a project (member repos stay in the app).
func (a *App) DeleteProject(id string) error {
	return a.engine.DeleteProject(id)
}

// AddProjectRepo adds a repo to a project.
func (a *App) AddProjectRepo(id, path string) error {
	return a.engine.AddProjectRepo(id, path)
}

// RemoveProjectRepo drops a repo from a project.
func (a *App) RemoveProjectRepo(id, path string) error {
	return a.engine.RemoveProjectRepo(id, path)
}

// SetProjectRepos replaces a project's membership order (drag-and-drop).
func (a *App) SetProjectRepos(id string, paths []string) error {
	return a.engine.SetProjectRepos(id, paths)
}

// FetchProject fetches every member repo. Returns repos fetched.
func (a *App) FetchProject(id string) (int, error) {
	return a.engine.FetchProject(id)
}

// ProjectCleanup bundles cleanup scan results for Wails binding
// (bindings support a single value plus error).
type ProjectCleanup struct {
	Candidates []*models.CleanupCandidate `json:"candidates"`
	Withheld   []*models.WithheldBranch   `json:"withheld"`
}

// GetProjectCleanup scans member repos for branches merged into base.
// Empty base resolves per repo (current branch).
func (a *App) GetProjectCleanup(id, base string) (*ProjectCleanup, error) {
	candidates, withheld, err := a.engine.ProjectCleanupScan(id, base)
	if err != nil {
		return nil, err
	}
	out := &ProjectCleanup{Candidates: []*models.CleanupCandidate{}, Withheld: []*models.WithheldBranch{}}
	for i := range candidates {
		out.Candidates = append(out.Candidates, &candidates[i])
	}
	for i := range withheld {
		out.Withheld = append(out.Withheld, &withheld[i])
	}
	return out, nil
}

// DeleteRepoBranch safe-deletes (-d) a branch in any repo path.
func (a *App) DeleteRepoBranch(repoPath, branch string) error {
	return a.engine.DeleteRepoBranch(repoPath, branch)
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
// API keys are masked — never sent in plaintext to the frontend.
func (a *App) GetAIConfig() (models.AIConfig, error) {
	cfg := a.engine.Config
	cfgDir := cfg.ConfigPath()

	// Gather per-provider key status
	providers := cfg.GetProviderKeys()
	ps := make([]models.ProviderStatus, 0, len(providers))
	for _, p := range providers {
		pc := cfg.GetProviderConfig(p)
		status := models.ProviderStatus{
			Provider: p,
			HasKey:   pc.APIKey != "",
			Model:    pc.Model,
			Endpoint: pc.Endpoint,
		}
		if pc.APIKey != "" {
			// Try decrypt to get last 4 chars for masking
			plaintext, err := ai.DecryptAPIKey(pc.APIKey, cfgDir)
			if err == nil && len(plaintext) >= 4 {
				prefix := plaintext[:min(4, len(plaintext))]
				if len(plaintext) > 4 {
					status.KeyMasked = prefix + "..." + plaintext[len(plaintext)-4:]
				} else {
					status.KeyMasked = prefix + "..." + plaintext
				}
			} else if err == nil {
				status.KeyMasked = "(saved)"
			} else {
				// Fallback: show stored format
				status.HasKey = false
				status.KeyMasked = ""
			}
		}
		ps = append(ps, status)
	}

	// Top-level config (provider may be set but key may be in per-provider storage)
	activeProvider := cfg.GetString("ai.provider")
	var activeKeyMasked string
	topLevelKey := cfg.GetString("ai.api_key")
	if topLevelKey != "" {
		plaintext, err := ai.DecryptAPIKey(topLevelKey, cfgDir)
		if err == nil {
			if len(plaintext) >= 8 {
				activeKeyMasked = plaintext[:4] + "..." + plaintext[len(plaintext)-4:]
			} else {
				activeKeyMasked = "(saved)"
			}
		}
	}
	// Also check per-provider config for the active provider
	if activeKeyMasked == "" {
		for _, p := range ps {
			if p.Provider == activeProvider {
				activeKeyMasked = p.KeyMasked
				break
			}
		}
	}

	return models.AIConfig{
		Provider:  activeProvider,
		APIKey:    activeKeyMasked,
		Model:     cfg.GetString("ai.model"),
		Endpoint:  cfg.GetString("ai.endpoint"),
		Providers: ps,
	}, nil
}

// GetUserPreferences returns the current user preferences (appearance + keybindings).
func (a *App) GetUserPreferences() models.UserPreferences {
	cfg := a.engine.Config.GetUserPreferences()
	return models.UserPreferences{
		Appearance: models.AppearanceConfig{
			Theme:       cfg.Appearance.Theme,
			AccentColor: cfg.Appearance.AccentColor,
			Brightness:  cfg.Appearance.Brightness,
		},
		Keybindings: cfg.Keybindings,
	}
}

// SetUserPreferences saves user preferences (appearance + keybindings) to disk.
func (a *App) SetUserPreferences(prefs models.UserPreferences) error {
	cfg := a.engine.Config
	cp := config.UserPreferences{
		Appearance: config.AppearanceConfig{
			Theme:       prefs.Appearance.Theme,
			AccentColor: prefs.Appearance.AccentColor,
			Brightness:  prefs.Appearance.Brightness,
		},
		Keybindings: prefs.Keybindings,
	}
	return cfg.SetUserPreferences(cp)
}

// SetAIConfig saves the AI provider configuration.
// The API key is encrypted before being stored in the config file.
func (a *App) SetAIConfig(provider, apiKey, model, endpoint string) error {
	cfg := a.engine.Config
	cfgDir := cfg.ConfigPath()

	if err := encryptAndSetProviderKey(cfg, cfgDir, provider, apiKey); err != nil {
		return err
	}

	cfg.Set("ai.provider", provider)
	cfg.Set("ai.model", model)
	cfg.Set("ai.endpoint", endpoint)

	// Also update the per-provider config
	pc := cfg.GetProviderConfig(provider)
	pc.Model = model
	if endpoint != "" {
		pc.Endpoint = endpoint
	}
	cfg.SetProviderConfig(provider, pc)

	return cfg.Save()
}

// SetProviderAIConfig saves config for a specific AI provider.
// The API key is encrypted before being stored.
func (a *App) SetProviderAIConfig(provider, apiKey, model, endpoint string) error {
	cfg := a.engine.Config
	cfgDir := cfg.ConfigPath()

	if err := encryptAndSetProviderKey(cfg, cfgDir, provider, apiKey); err != nil {
		return err
	}

	cfg.SetProviderConfig(provider, config.ProviderConfig{
		APIKey:   cfg.GetString("ai.providers." + provider + ".api_key"),
		Model:    model,
		Endpoint: endpoint,
	})

	// If this is the active provider, also update top-level fields
	if cfg.GetString("ai.provider") == provider {
		cfg.Set("ai.model", model)
		if endpoint != "" {
			cfg.Set("ai.endpoint", endpoint)
		}
	}

	return cfg.Save()
}

// encryptAndSetProviderKey encrypts and stores the API key for the given provider.
// If apiKey is empty, it deletes the stored key.
func encryptAndSetProviderKey(cfg *config.Manager, cfgDir, provider, apiKey string) error {
	if apiKey == "" {
		// Clear the key for this provider
		cfg.Set("ai.providers."+provider+".api_key", "")
		cfg.Set("ai.api_key", "")
		return nil
	}

	encrypted, err := ai.EncryptAPIKey(apiKey, cfgDir)
	if err != nil {
		return fmt.Errorf("encrypt API key: %w", err)
	}

	// Store encrypted key both top-level (for backward compat) and per-provider
	cfg.Set("ai.api_key", encrypted)
	cfg.Set("ai.providers."+provider+".api_key", encrypted)
	return nil
}

// DeleteProviderAIConfig removes all config for a specific provider (key, model, endpoint).
func (a *App) DeleteProviderAIConfig(provider string) error {
	cfg := a.engine.Config
	cfg.DeleteProviderConfig(provider)

	// If this was the active provider, also clear top-level AI fields
	if cfg.GetString("ai.provider") == provider {
		cfg.Set("ai.provider", "")
		cfg.Set("ai.api_key", "")
		cfg.Set("ai.model", "")
		cfg.Set("ai.endpoint", "")
	}

	return cfg.Save()
}

// FetchProviderModels queries the provider's /models API and returns available model IDs.
// apiKey is required for OpenAI, OpenRouter, Groq; optional for Ollama.
func (a *App) FetchProviderModels(provider, apiKey string) ([]string, error) {
	// If no apiKey provided, try the stored encrypted key
	if apiKey == "" {
		cfg := a.engine.Config
		cfgDir := cfg.ConfigPath()
		pc := cfg.GetProviderConfig(provider)
		if pc.APIKey != "" {
			decrypted, err := ai.DecryptAPIKey(pc.APIKey, cfgDir)
			if err == nil {
				apiKey = decrypted
			}
		}
	}

	models, err := ai.FetchProviderModels(provider, apiKey)
	if err != nil {
		return nil, err
	}
	// Cache the result for offline use
	if err := a.cacheProviderModels(provider, models); err != nil {
		log.Printf("zgit: cache provider models %s: %v", provider, err)
	}
	return models, nil
}

// GetCachedProviderModels returns cached provider models if fresh,
// otherwise fetches from the provider API and caches the result.
func (a *App) GetCachedProviderModels(provider string) ([]string, error) {
	scope := "ai_models:" + provider
	key := "models"
	ttl := ai.ProviderModelCacheTTL

	// Try cache first
	data, ok := a.engine.Store.GetCacheEntry(scope, key, ttl)
	if ok {
		var cached []string
		if err := json.Unmarshal(data, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	// Cache miss — fetch live
	models, err := ai.FetchProviderModels(provider, "")
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := a.cacheProviderModels(provider, models); err != nil {
		log.Printf("zgit: cache provider models %s: %v", provider, err)
	}

	return models, nil
}

// cacheProviderModels stores the model list in the SQLite cache.
func (a *App) cacheProviderModels(provider string, models []string) error {
	scope := "ai_models:" + provider
	key := "models"
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	return a.engine.Store.SetCacheEntry(scope, key, data)
}

// TestProviderConnection checks if a provider's API is reachable.
// Returns (ok, latencyMs, error).
func (a *App) TestProviderConnection(provider string) (bool, int64, error) {
	cfg := a.engine.Config
	cfgDir := cfg.ConfigPath()
	pc := cfg.GetProviderConfig(provider)
	apiKey := pc.APIKey
	if apiKey != "" {
		decrypted, err := ai.DecryptAPIKey(apiKey, cfgDir)
		if err == nil {
			apiKey = decrypted
		}
	}

	start := time.Now()
	_, err := ai.FetchProviderModels(provider, apiKey)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		return false, latencyMs, err
	}
	return true, latencyMs, nil
}

// AIError categorizes AI provider errors for user-friendly recovery.
type AIError struct {
	Category       string // "auth", "rate_limit", "context_length", "network", "provider"
	Retryable      bool
	SuggestedModel string // fallback model to try
	Message        string
}

func (e *AIError) Error() string {
	return e.Message
}

// categorizeError inspects an error message and returns a typed AIError.
func categorizeError(err error) *AIError {
	msg := err.Error()
	if msg == "" {
		return &AIError{Category: "provider", Retryable: true, Message: "unknown AI error"}
	}
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "401") || strings.Contains(lower, "403") || strings.Contains(lower, "invalid api key") || strings.Contains(lower, "authentication"):
		return &AIError{Category: "auth", Retryable: false, Message: "API key invalid or unauthorized — check your API key in Settings"}
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "too many requests"):
		return &AIError{Category: "rate_limit", Retryable: true, Message: "API rate limit exceeded — please wait and retry"}
	case strings.Contains(lower, "context length") || strings.Contains(lower, "maximum context") || strings.Contains(lower, "token limit"):
		return &AIError{Category: "context_length", Retryable: true, SuggestedModel: "claude-3-haiku-20240307", Message: "Input too long for model context window"}
	case strings.Contains(lower, "connection") || strings.Contains(lower, "timeout") || strings.Contains(lower, "network") || strings.Contains(lower, "unreachable"):
		return &AIError{Category: "network", Retryable: true, Message: "Network error — check your internet connection"}
	default:
		return &AIError{Category: "provider", Retryable: true, Message: msg}
	}
}

// GenerateCommitMessage reads the staged diff and returns an AI-generated conventional commit message.
func (a *App) GenerateCommitMessage() (string, error) {
	return a.GenerateCommitMessageWithContext()
}

// GenerateCommitMessageWithContext generates a commit message using the staged diff
// plus repository context (branch, recent commits, status) to match project conventions.
// On context_length errors, auto-falls back to a smaller model.
func (a *App) GenerateCommitMessageWithContext() (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()

	aiCfg := a.aiConfig()
	if aiCfg.Provider == "" {
		return "", fmt.Errorf("AI provider not configured — go to Settings to set up AI commit message generation")
	}
	if aiCfg.APIKey == "" {
		return "", fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	msg, err := a.engine.GenerateCommitMessageWithContext(ctx, aiCfg)
	if err != nil {
		aiErr := categorizeError(err)
		// Auto-fallback on context_length errors
		if aiErr.Category == "context_length" && aiErr.SuggestedModel != "" && aiCfg.Model != aiErr.SuggestedModel {
			fallbackCfg := aiCfg
			fallbackCfg.Model = aiErr.SuggestedModel
			msg, fbErr := a.engine.GenerateCommitMessageWithContext(ctx, fallbackCfg)
			if fbErr == nil {
				return msg, nil
			}
			// If fallback also fails, return the original error with fallback info
			return "", &AIError{
				Category:       aiErr.Category,
				Retryable:      aiErr.Retryable,
				SuggestedModel: aiErr.SuggestedModel,
				Message:        aiErr.Message + " (fallback to " + aiErr.SuggestedModel + " also failed)",
			}
		}
		return "", aiErr
	}

	return msg, nil
}

// GenerateCommitMessageStream streams a commit message via onToken callback.
// Returns the full message when complete.
func (a *App) GenerateCommitMessageStream(onToken func(string)) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 120e9)
	defer cancel()

	aiCfg := a.aiConfig()
	if aiCfg.Provider == "" {
		return "", fmt.Errorf("AI provider not configured — go to Settings to set up AI commit message generation")
	}
	if aiCfg.APIKey == "" {
		return "", fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	msg, err := a.engine.GenerateCommitMessageWithContextStream(ctx, aiCfg, onToken)
	if err != nil {
		aiErr := categorizeError(err)
		return "", aiErr
	}

	return msg, nil
}

// GeneratePRDescription generates a PR title and description from the diff between head and base branches.
func (a *App) GeneratePRDescription(head, base string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 120e9)
	defer cancel()

	if head == "" {
		return "", fmt.Errorf("head branch is required")
	}
	if base == "" {
		base = "main"
	}

	// Get diff between base and head
	diff, err := a.engine.Git.Diff(ctx, git.DiffOptions{A: base, B: head, Unified: true})
	if err != nil {
		return "", fmt.Errorf("get branch diff: %w", err)
	}
	if diff == nil || len(diff.Files) == 0 {
		return "", fmt.Errorf("no differences between %s and %s", base, head)
	}

	summary := ai.SummarizeDiff(diff)
	diffText := summary.Summary

	// Get recent commits for context
	var logContext string
	log, logErr := a.engine.Git.Log(ctx, git.LogOptions{Count: 10, Branch: head})
	if logErr == nil && len(log) > 0 {
		var sb strings.Builder
		sb.WriteString("\nRecent commits on " + head + ":\n")
		for _, c := range log {
			sb.WriteString("  " + c.Hash[:7] + " " + c.Message + "\n")
		}
		logContext = sb.String()
	}

	diffText = fmt.Sprintf("Stats: %d files changed (%d lockfile/binary filtered).\n%s\n\n%s",
		summary.ChangedFiles, summary.Filtered, logContext, diffText)

	aiCfg := a.aiConfig()
	if aiCfg.Provider == "" {
		return "", fmt.Errorf("AI provider not configured — go to Settings first")
	}
	if aiCfg.APIKey == "" {
		return "", fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	generator, err := ai.NewGenerator(aiCfg)
	if err != nil {
		return "", err
	}

	prompt := `You are a PR description generator. Create a pull request title and structured description.

Format:
PR Title: <concise title following conventional commits>

## Summary
<2-3 sentence overview of what this PR does>

## Key Changes
- <change 1>
- <change 2>
- <change 3>

## Test Plan
<suggestions for testing>

Return ONLY the PR title and description. No extra commentary.`

	msg, err := generator.GenerateCommitMessage(ctx, fmt.Sprintf("%s\n\nBranch diff to review:\n\n```\n%s\n```", prompt, diffText), aiCfg)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	return msg, nil
}

// GeneratePRMetadata generates a PR title and description from the diff between base and head branches.
// Uses three-dot diff (base...head) to show only commits unique to head.
// Returns structured output: "PR Title: ..." followed by description body.
func (a *App) GeneratePRMetadata(base, head string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 120e9)
	defer cancel()

	if head == "" {
		return "", fmt.Errorf("head branch is required")
	}
	if base == "" {
		base = "main"
	}

	// Three-dot diff: shows only commits in head not in base
	diff, err := a.engine.Git.Diff(ctx, git.DiffOptions{
		A:       base + "..." + head,
		Unified: true,
	})
	if err != nil {
		return "", fmt.Errorf("get branch diff: %w", err)
	}
	if diff == nil || len(diff.Files) == 0 {
		return "", fmt.Errorf("no differences between %s and %s", base, head)
	}

	summary := ai.SummarizeDiff(diff)
	diffText := summary.Summary

	// Get recent commits for richer context
	var logContext string
	log, logErr := a.engine.Git.Log(ctx, git.LogOptions{Count: 10, Branch: head})
	if logErr == nil && len(log) > 0 {
		var sb strings.Builder
		sb.WriteString("\nRecent commits on " + head + ":\n")
		for _, c := range log {
			sb.WriteString("  " + c.Hash[:7] + " " + c.Message + "\n")
		}
		logContext = sb.String()
	}

	diffText = fmt.Sprintf("Stats: %d files changed (%d lockfile/binary filtered).\n%s\n\n%s",
		summary.ChangedFiles, summary.Filtered, logContext, diffText)

	aiCfg := a.aiConfig()
	if aiCfg.Provider == "" {
		return "", fmt.Errorf("AI provider not configured — go to Settings first")
	}
	if aiCfg.APIKey == "" {
		return "", fmt.Errorf("API key not configured — go to Settings to set your API key")
	}

	generator, err := ai.NewGenerator(aiCfg)
	if err != nil {
		return "", err
	}

	prompt := `Act as a Senior Developer. Summarize this diff into a professional PR Title and a structured Description with bullet points.

Format:
PR Title: <concise title following conventional commits>

## Summary
<2-3 sentence overview of what this PR does and why>

## Key Changes
- <change 1 description>
- <change 2 description>
- <change 3 description>

## Test Plan
<suggestions for testing>

Return ONLY the PR title and description. No extra commentary.`

	msg, err := generator.GenerateCommitMessage(ctx, fmt.Sprintf("%s\n\nBranch diff to review:\n\n```\n%s\n```", prompt, diffText), aiCfg)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	return msg, nil
}

// AIGetRepoContext returns a formatted snapshot of the current repository state
// for use by AI slash commands and the agent.
func (a *App) AIGetRepoContext() (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 30*time.Second)
	defer cancel()

	var sb strings.Builder

	// Branch
	branch, err := a.engine.Git.CurrentBranch(ctx)
	if err == nil {
		sb.WriteString("Branch: " + branch + "\n")
	}

	// Status
	status, err := a.engine.Git.Status(ctx)
	if err == nil {
		sb.WriteString(fmt.Sprintf("Status: %d staged, %d unstaged, %d untracked\n",
			status.StagedCount, status.UnstagedCount, status.UntrackedCount))
		sb.WriteString(fmt.Sprintf("Clean: %v, Merging: %v, Rebasing: %v\n",
			status.IsClean, status.IsMerging, status.IsRebasing))
	}

	// Recent log (last 10)
	log, err := a.engine.Git.Log(ctx, git.LogOptions{Count: 10})
	if err == nil {
		sb.WriteString("\nRecent commits:\n")
		for _, c := range log {
			sb.WriteString(fmt.Sprintf("  %s %s\n", c.Hash[:7], c.Message))
		}
	}

	// Unstaged diffs (truncated)
	diff, err := a.engine.Git.Diff(ctx, git.DiffOptions{Unified: true})
	if err == nil && len(diff.Files) > 0 {
		sb.WriteString("\nUnstaged changes:\n")
		summary := ai.SummarizeDiff(diff)
		sb.WriteString(summary.Summary)
	}

	// Conflict files
	conflicts, err := a.engine.Git.ConflictFiles(ctx)
	if err == nil && len(conflicts) > 0 {
		sb.WriteString("\nMerge conflicts:\n")
		for _, cf := range conflicts {
			sb.WriteString(fmt.Sprintf("  %s (%d blocks)\n", cf.Path, cf.BlockCount))
		}
	}

	return sb.String(), nil
}

// AIReviewPR returns the diff between source and target branches for code review.
func (a *App) AIReviewPR(source, target string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 60*time.Second)
	defer cancel()

	if source == "" {
		return "", fmt.Errorf("source branch is required")
	}
	if target == "" {
		target = "main"
	}

	diff, err := a.engine.Git.Diff(ctx, git.DiffOptions{
		A:       target,
		B:       source,
		Unified: true,
	})
	if err != nil {
		return "", fmt.Errorf("get branch diff: %w", err)
	}
	if diff == nil || len(diff.Files) == 0 {
		return "No differences between " + target + " and " + source, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Diff between %s (base) and %s (source):\n", target, source))
	sb.WriteString(fmt.Sprintf("Total: %d files, +%d / -%d lines\n\n",
		len(diff.Files), diff.TotalAdds, diff.TotalDeletes))

	summary := ai.SummarizeDiff(diff)
	sb.WriteString(summary.Summary)

	return sb.String(), nil
}

// aiConfig extracts AI provider config from the engine settings.
func (a *App) aiConfig() ai.Config {
	cfg := a.engine.Config
	cfgDir := cfg.ConfigPath()
	provider := ai.ProviderKind(cfg.GetString("ai.provider"))
	// Try to decrypt the top-level API key; fall back to per-provider key if decryption fails
	var apiKey string
	topLevelKey := cfg.GetString("ai.api_key")
	if topLevelKey != "" {
		plaintext, err := ai.DecryptAPIKey(topLevelKey, cfgDir)
		if err == nil {
			apiKey = plaintext
		}
	}
	// If top-level decryption failed, try per-provider key
	if apiKey == "" {
		pc := cfg.GetProviderConfig(string(provider))
		if pc.APIKey != "" {
			plaintext, err := ai.DecryptAPIKey(pc.APIKey, cfgDir)
			if err == nil {
				apiKey = plaintext
			}
		}
	}
	return ai.Config{
		Provider: provider,
		APIKey:   apiKey,
		Model:    cfg.GetString("ai.model"),
		Endpoint: cfg.GetString("ai.endpoint"),
	}
}

// --- Agentic AI Assistant ---

// AgentStart creates a new agent session with the configured provider.
func (a *App) AgentStart() error {
	a.agentMu.Lock()
	defer a.agentMu.Unlock()

	aiCfg := a.aiConfig()
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
	cfg := a.engine.Config
	cfgDir := cfg.ConfigPath()
	provider := ai.ProviderKind(cfg.GetString("ai.provider"))
	// Try to decrypt the top-level API key; fall back to per-provider key if decryption fails
	var apiKey string
	topLevelKey := cfg.GetString("ai.api_key")
	if topLevelKey != "" {
		plaintext, err := ai.DecryptAPIKey(topLevelKey, cfgDir)
		if err == nil {
			apiKey = plaintext
		}
	}
	// If top-level decryption failed, try per-provider key
	if apiKey == "" {
		pc := cfg.GetProviderConfig(string(provider))
		if pc.APIKey != "" {
			plaintext, err := ai.DecryptAPIKey(pc.APIKey, cfgDir)
			if err == nil {
				apiKey = plaintext
			}
		}
	}
	return ai.Config{
		Provider: provider,
		APIKey:   apiKey,
		Model:    cfg.GetString("ai.model"),
		Endpoint: cfg.GetString("ai.endpoint"),
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
		wailsRuntime.EventsEmit(a.ctx, "ai:error", "AI provider not configured — go to Settings to set up AI")
		return nil
	}
	if aiCfg.APIKey == "" {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", "API key not configured — go to Settings to set your API key")
		return nil
	}

	provider, err := ai.NewAskProvider(aiCfg)
	if err != nil {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("create ask provider: %s", err.Error()))
		return nil
	}

	// Ensure active Ask session
	session := a.sessionManager.Active()
	if session == nil || session.Mode != "ask" {
		session, err = a.sessionManager.Create("Ask Session", "ask")
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("create session: %s", err.Error()))
			return nil
		}
	}

	// Add user message
	if err := a.sessionManager.AddMessage(session.ID, ai.Message{Role: "user", Content: message}); err != nil {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("add message: %s", err.Error()))
		return nil
	}

	messages, err := a.sessionManager.GetMessages(session.ID)
	if err != nil {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("get messages: %s", err.Error()))
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
			wailsRuntime.EventsEmit(a.ctx, "ai:token", token)
		})
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, "ai:error", err.Error())
			return
		}

		// Add assistant response to session
		if err := a.sessionManager.AddMessage(session.ID, resp); err != nil {
			wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("save response: %s", err.Error()))
			return
		}

		wailsRuntime.EventsEmit(a.ctx, "ai:done", resp.Content)
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

// --- AI Explain ---

// explainRequest carries the diff + context the AI should explain.
type explainRequest struct {
	Diff    string `json:"diff"`
	Context string `json:"context"` // e.g. commit message, file path, refs
	Kind    string `json:"kind"`    // "commit", "filediff", "selection"
}

// buildExplainPrompt constructs the system-ish prompt prepended to the user message.
func buildExplainPrompt(req explainRequest) string {
	var sb strings.Builder
	sb.WriteString("You are a helpful coding assistant explaining Git changes. Explain the following diff clearly and concisely.\n\n")
	sb.WriteString("Focus on:\n")
	sb.WriteString("- What changed and why it matters\n")
	sb.WriteString("- Whether the change is likely safe or risky\n")
	sb.WriteString("- Any obvious issues (bugs, missing tests, security concerns)\n\n")
	sb.WriteString("Format:\n")
	sb.WriteString("## Summary\n<2-3 sentence overview>\n\n")
	sb.WriteString("## Key Changes\n- <change 1>\n- <change 2>\n\n")
	sb.WriteString("## Notes\n<optional cautions or suggestions>\n\n")
	sb.WriteString("Return only the explanation. No extra commentary.\n\n")
	sb.WriteString("Diff context: " + req.Context + "\n\n")
	sb.WriteString("```diff\n" + req.Diff + "\n```\n")
	return sb.String()
}

// explainWithAI runs the diff through the AI ask provider and returns the explanation.
func (a *App) explainWithAI(req explainRequest) (string, error) {
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

	session := a.sessionManager.Active()
	if session == nil || session.Mode != "ask" {
		session, err = a.sessionManager.Create("Explain Session", "ask")
		if err != nil {
			return "", fmt.Errorf("create session: %w", err)
		}
	}

	prompt := buildExplainPrompt(req)
	if err := a.sessionManager.AddMessage(session.ID, ai.Message{Role: "user", Content: prompt}); err != nil {
		return "", fmt.Errorf("add message: %w", err)
	}

	messages, err := a.sessionManager.GetMessages(session.ID)
	if err != nil {
		return "", fmt.Errorf("get messages: %w", err)
	}

	resp, err := provider.Ask(a.getContext(), messages)
	if err != nil {
		return "", fmt.Errorf("explain failed: %w", err)
	}

	if err := a.sessionManager.AddMessage(session.ID, resp); err != nil {
		return "", fmt.Errorf("add response: %w", err)
	}

	return resp.Content, nil
}

// ExplainCommit explains a single commit: its message + diff.
func (a *App) ExplainCommit(sha string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()

	if sha == "" {
		return "", fmt.Errorf("commit SHA is required")
	}

	// Get the commit message for context.
	commits, err := a.engine.Git.Log(ctx, git.LogOptions{Branch: sha, Count: 1})
	if err != nil {
		return "", fmt.Errorf("get commit: %w", err)
	}
	context := "Commit " + sha[:7]
	if len(commits) > 0 {
		context = fmt.Sprintf("Commit %s (%s)\nMessage: %s", sha[:7], commits[0].Author, commits[0].Message)
	}

	// Get the commit diff (parent..commit). Root commits have no parent.
	var diffText string
	diff, err := a.engine.Git.Diff(ctx, git.DiffOptions{A: sha + "^", B: sha, Unified: true})
	if err != nil {
		// Root commit or no parent — explain from message only.
		diffText = "(no parent diff available — root commit)"
	} else {
		diffText = diffToString(diff)
	}

	return a.explainWithAI(explainRequest{
		Kind:    "commit",
		Context: context,
		Diff:    diffText,
	})
}

// ExplainFileDiff explains the diff of a file between two refs.
func (a *App) ExplainFileDiff(path, ref1, ref2 string) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()

	if path == "" {
		return "", fmt.Errorf("file path is required")
	}
	if ref1 == "" {
		ref1 = "HEAD~1"
	}
	if ref2 == "" {
		ref2 = "HEAD"
	}

	diff, err := a.engine.Git.Diff(ctx, git.DiffOptions{A: ref1, B: ref2, Pathspec: path, Unified: true})
	if err != nil {
		return "", fmt.Errorf("get file diff: %w", err)
	}

	context := fmt.Sprintf("File: %s\nRange: %s..%s", path, ref1, ref2)
	return a.explainWithAI(explainRequest{
		Kind:    "filediff",
		Context: context,
		Diff:    diffToString(diff),
	})
}

// ExplainSelection explains arbitrary diff text provided by the frontend.
func (a *App) ExplainSelection(diffText string) (string, error) {
	if strings.TrimSpace(diffText) == "" {
		return "", fmt.Errorf("no diff text to explain")
	}
	return a.explainWithAI(explainRequest{
		Kind:    "selection",
		Context: "User-selected diff",
		Diff:    diffText,
	})
}

// diffToString flattens a models.Diff into a readable unified-diff string.
func diffToString(d *models.Diff) string {
	if d == nil || len(d.Files) == 0 {
		return "(no changes)"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total: %d files, +%d / -%d lines\n\n", len(d.Files), d.TotalAdds, d.TotalDeletes))
	for _, f := range d.Files {
		b.WriteString(fmt.Sprintf("File: %s (%s, +%d / -%d)\n", f.NewPath, changeType(f.Type), f.Additions, f.Deletions))
		if f.UnifiedDiff != "" {
			b.WriteString(f.UnifiedDiff)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// changeType returns a short label for a file change type.
func changeType(t models.FileChangeType) string {
	switch t {
	case models.FileAdded:
		return "added"
	case models.FileModified:
		return "modified"
	case models.FileDeleted:
		return "deleted"
	case models.FileRenamed:
		return "renamed"
	case models.FileCopied:
		return "copied"
	default:
		return "changed"
	}
}

// ExplainStream streams an explanation token-by-token via ai:token events.
func (a *App) ExplainStream(message string) error {
	aiCfg := a.getAskConfig()
	if aiCfg.Provider == "" {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", "AI provider not configured — go to Settings to set up AI")
		return nil
	}
	if aiCfg.APIKey == "" {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", "API key not configured — go to Settings to set your API key")
		return nil
	}

	provider, err := ai.NewAskProvider(aiCfg)
	if err != nil {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("create ask provider: %s", err.Error()))
		return nil
	}

	session := a.sessionManager.Active()
	if session == nil || session.Mode != "ask" {
		session, err = a.sessionManager.Create("Explain Stream Session", "ask")
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("create session: %s", err.Error()))
			return nil
		}
	}

	if err := a.sessionManager.AddMessage(session.ID, ai.Message{Role: "user", Content: message}); err != nil {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("add message: %s", err.Error()))
		return nil
	}

	messages, err := a.sessionManager.GetMessages(session.ID)
	if err != nil {
		wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("get messages: %s", err.Error()))
		return nil
	}

	ctx, cancel := context.WithCancel(a.getContext())
	a.askCancelMu.Lock()
	if a.askCancel != nil {
		a.askCancel()
	}
	a.askCancel = cancel
	a.askCancelMu.Unlock()

	go func() {
		defer cancel()

		resp, err := provider.AskStream(ctx, messages, func(token string) {
			wailsRuntime.EventsEmit(a.ctx, "ai:token", token)
		})
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, "ai:error", err.Error())
			return
		}

		if err := a.sessionManager.AddMessage(session.ID, resp); err != nil {
			wailsRuntime.EventsEmit(a.ctx, "ai:error", fmt.Sprintf("save response: %s", err.Error()))
			return
		}

		wailsRuntime.EventsEmit(a.ctx, "ai:done", resp.Content)
	}()

	return nil
}

// --- GitHub PR & Issue Hub ---

// githubRepo returns the owner/name of the currently open repo, or an error
// if no repo is open or it has no GitHub metadata.
func (a *App) githubRepo() (string, string, error) {
	repo := a.engine.CurrentRepo()
	if repo == nil || repo.Owner == "" || repo.Name == "" {
		return "", "", fmt.Errorf("no GitHub repo detected for the current repository")
	}
	return repo.Owner, repo.Name, nil
}

// requireGitHub returns the GitHub client, erroring if not authenticated.
func (a *App) requireGitHub() (github.GitHubClient, error) {
	if a.engine.GitHub == nil {
		return nil, fmt.Errorf("GitHub not authenticated — set a token in Settings first")
	}
	return a.engine.GitHub, nil
}

// GitHubListPullRequests lists PRs for the current repo.
func (a *App) GitHubListPullRequests(state string) ([]*models.PullRequestSummary, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	opts := github.PRFilter{State: state, Limit: 50}
	return gh.ListPullRequests(ctx, owner, repo, opts)
}

// GitHubGetPullRequest returns a single PR by number.
func (a *App) GitHubGetPullRequest(number int) (*models.PullRequestDetail, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return gh.GetPullRequest(ctx, owner, repo, number)
}

// GitHubCreatePullRequest creates a PR from head onto base.
func (a *App) GitHubCreatePullRequest(title, body, head, base string, draft bool) (*models.PullRequestSummary, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return gh.CreatePullRequest(ctx, owner, repo, github.PRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
		Draft: draft,
	})
}

// GitHubMergePullRequest merges a PR by number.
func (a *App) GitHubMergePullRequest(number int, method string) error {
	gh, err := a.requireGitHub()
	if err != nil {
		return err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return gh.MergePullRequest(ctx, owner, repo, number, method)
}

// GitHubListIssues lists issues for the current repo.
func (a *App) GitHubListIssues(state string) ([]*models.Issue, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	opts := github.IssuesFilter{State: state, Limit: 50}
	return gh.ListIssues(ctx, owner, repo, opts)
}

// GitHubCreateIssue creates an issue in the current repo.
func (a *App) GitHubCreateIssue(title, body string) (*models.Issue, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return gh.CreateIssue(ctx, owner, repo, github.IssueRequest{Title: title, Body: body})
}

// GitHubCloseIssue closes an issue by number.
func (a *App) GitHubCloseIssue(number int) error {
	gh, err := a.requireGitHub()
	if err != nil {
		return err
	}
	owner, repo, err := a.githubRepo()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return gh.CloseIssue(ctx, owner, repo, number)
}

// GitHubListNotifications lists the authenticated user's GitHub notifications.
func (a *App) GitHubListNotifications(all bool) ([]*models.Notification, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return gh.ListNotifications(ctx, all, 50)
}

// GitHubGetDashboard returns the GitHub dashboard (my PRs, review PRs, issues, runs).
func (a *App) GitHubGetDashboard() (*models.Dashboard, error) {
	gh, err := a.requireGitHub()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 30e9)
	defer cancel()
	return gh.GetDashboard(ctx)
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
	if err := a.sessionManager.Rename(id, name); err != nil {
		return err
	}
	return nil
}

// SessionDelete deletes a session.
func (a *App) SessionDelete(id string) error {
	if err := a.sessionManager.Delete(id); err != nil {
		return err
	}
	return nil
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
	if err := a.sessionManager.ClearMessages(id); err != nil {
		return err
	}
	return nil
}

// SessionSetMode changes the mode of a session ("ask" or "agent").
func (a *App) SessionSetMode(id, mode string) error {
	if err := a.sessionManager.SetMode(id, mode); err != nil {
		return err
	}
	return nil
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

// SaveGitHubToken saves a GitHub token without validating it.
// Returns immediately — no network calls. Use ValidateGitHubToken separately.
func (a *App) SaveGitHubToken(token string) error {
	return a.engine.SaveGitHubToken(token)
}

// ValidateGitHubToken fetches the authenticated GitHub user to validate the token.
// Call this after SaveGitHubToken if you need the user profile.
// Returns the user on success, or an error if validation fails.
func (a *App) ValidateGitHubToken() (*models.User, error) {
	if !a.engine.IsGitHubAuthenticated() {
		return nil, fmt.Errorf("GitHub not authenticated — save a token first")
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 10e9)
	defer cancel()
	return a.engine.GitHub.GetAuthenticatedUser(ctx)
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

	// github.PollDeviceFlow returns ("", nil) for authorization_pending internally.
	return github.PollDeviceFlow(ctx, clientID, deviceCode)
}

// PollDeviceFlowWithRetry polls GitHub for device flow authorization with automatic retry.
// It respects the poll interval, handles slow_down by increasing interval, and blocks until
// the user authorizes, the device code expires, or 10 minutes elapses.
// The interval parameter comes from the StartDeviceFlow response.
func (a *App) PollDeviceFlowWithRetry(deviceCode string, interval int) (string, error) {
	// Create cancellable context — CancelDeviceFlow can stop this early
	ctx, cancel := context.WithCancel(a.getContext())
	a.devicePollMu.Lock()
	if a.devicePollCancel != nil {
		a.devicePollCancel()
	}
	a.devicePollCancel = cancel
	a.devicePollMu.Unlock()

	clientID := a.engine.Config.GetString("github.device_flow_client_id")
	if clientID == "" {
		clientID = github.DefaultDeviceFlowClientID
	}

	token, err := github.PollDeviceFlowWithRetry(ctx, clientID, deviceCode, interval)
	// Clean up cancel func after poll completes
	a.devicePollMu.Lock()
	a.devicePollCancel = nil
	a.devicePollMu.Unlock()
	return token, err
}

// CancelDeviceFlow cancels an in-flight PollDeviceFlowWithRetry call.
func (a *App) CancelDeviceFlow() {
	a.devicePollMu.Lock()
	defer a.devicePollMu.Unlock()
	if a.devicePollCancel != nil {
		a.devicePollCancel()
		a.devicePollCancel = nil
	}
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

// LogoutGitHub clears the GitHub token and client.
func (a *App) LogoutGitHub() error {
	return a.engine.LogoutGitHub()
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

// GetIssues lists issues with the given state ("open", "closed", "all").
// Empty state defaults to "open".
func (a *App) GetIssues(state string) ([]*models.Issue, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return []*models.Issue{}, nil
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return []*models.Issue{}, nil
	}
	if state != "open" && state != "closed" && state != "all" {
		state = "open"
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	issues, err := gh.ListIssues(ctx, owner, repo, github.IssuesFilter{
		State: state,
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
	// Track in the store (recency) and the legacy viper list.
	a.engine.TrackRepo(resolved)
	// Restart file watcher for the new repo
	a.restartFileWatcher()
	// Notify frontend
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "repo:switched", resolved)
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
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
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

// --- Notifications (global inbox, no repo scope) ---

// GetNotifications returns inbox entries, newest first.
// all=false returns unread only.
func (a *App) GetNotifications(all bool) ([]*models.Notification, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return []*models.Notification{}, nil
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.ListNotifications(ctx, all, 50)
}

// MarkNotificationRead marks one thread read.
func (a *App) MarkNotificationRead(threadID string) error {
	gh, err := a.getGitHubClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.MarkNotificationRead(ctx, threadID)
}

// MarkAllNotificationsRead marks the whole inbox read.
func (a *App) MarkAllNotificationsRead() error {
	gh, err := a.getGitHubClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 15e9)
	defer cancel()
	return gh.MarkAllNotificationsRead(ctx)
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

// CheckoutPR fetches pull/N/head and checks out a local pr/N branch.
// Returns the local branch name.
func (a *App) CheckoutPR(number int) (string, error) {
	ctx, cancel := context.WithTimeout(a.getContext(), 60e9)
	defer cancel()
	return a.engine.CheckoutPR(ctx, number)
}

// PRChecks aggregates check runs + workflow runs for a PR's head commit.
type PRChecks struct {
	Number  int                   `json:"number"`
	HeadSHA string                `json:"head_sha"`
	HeadRef string                `json:"head_ref"`
	Checks  []*models.CheckRun    `json:"checks"`
	Runs    []*models.WorkflowRun `json:"runs"`
}

// GetPRChecks returns CI checks and workflow runs for a pull request.
func (a *App) GetPRChecks(number int) (*PRChecks, error) {
	gh, err := a.getGitHubClient()
	if err != nil {
		return nil, err
	}
	owner, repo, err := a.guessRepo()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.getContext(), 20e9)
	defer cancel()
	detail, err := gh.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	out := &PRChecks{Number: number, HeadRef: detail.HeadRef, HeadSHA: detail.HeadSHA}
	ref := detail.HeadSHA
	if ref == "" {
		ref = detail.HeadRef
	}
	if ref == "" {
		out.Checks = []*models.CheckRun{}
		out.Runs = []*models.WorkflowRun{}
		return out, nil
	}
	checks, err := gh.ListCheckRunsForRef(ctx, owner, repo, ref)
	if err != nil {
		return nil, err
	}
	out.Checks = checks
	if out.Checks == nil {
		out.Checks = []*models.CheckRun{}
	}
	if detail.HeadSHA != "" {
		runs, err := gh.ListWorkflowRuns(ctx, owner, repo, github.RunsFilter{
			HeadSHA: detail.HeadSHA,
			Limit:   10,
		})
		if err != nil {
			return nil, err
		}
		out.Runs = runs
	}
	if out.Runs == nil {
		out.Runs = []*models.WorkflowRun{}
	}
	return out, nil
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
