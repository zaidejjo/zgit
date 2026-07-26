package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// NativeExec implements GitAdapter by shelling out to the system git binary.
// It caches the git path at initialization and uses os/exec for all operations.
// All methods are safe for concurrent use.
type NativeExec struct {
	mu       sync.RWMutex
	repoPath string
	gitPath  string // resolved `git` binary path
}

// NewNativeExec creates a new NativeExec by resolving the git binary path.
// gitPath can be empty to use PATH resolution.
func NewNativeExec(gitPath string) (*NativeExec, error) {
	if gitPath == "" {
		var err error
		gitPath, err = exec.LookPath("git")
		if err != nil {
			return nil, fmt.Errorf("git not found in PATH: %w", err)
		}
	}
	return &NativeExec{gitPath: gitPath}, nil
}

// Open sets the repository path. All subsequent git commands run in this directory.
func (n *NativeExec) Open(path string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.repoPath = path
	return nil
}

// Close clears the repository path.
func (n *NativeExec) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.repoPath = ""
	return nil
}

// Path returns the currently open repository path.
func (n *NativeExec) Path() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.repoPath
}

// run executes a git command and returns stdout as bytes.
func (n *NativeExec) run(ctx context.Context, args ...string) ([]byte, error) {
	n.mu.RLock()
	repoPath := n.repoPath
	n.mu.RUnlock()

	if repoPath == "" {
		return nil, fmt.Errorf("no repository open: call Open() first")
	}

	cmd := exec.CommandContext(ctx, n.gitPath, args...)
	cmd.Dir = repoPath

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return nil, &GitError{
			Stderr:   stderr.String(),
			Args:     args,
			ExitCode: exitCode,
		}
	}

	return stdout.Bytes(), nil
}

// runInDir executes a git command in the specified directory (for operations outside repo).
func (n *NativeExec) runInDir(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, n.gitPath, args...)
	cmd.Dir = dir

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return nil, &GitError{
			Stderr:   stderr.String(),
			Args:     args,
			ExitCode: exitCode,
		}
	}

	return stdout.Bytes(), nil
}

// stream executes a git command and calls handler for each output line.
func (n *NativeExec) stream(ctx context.Context, handler func(line []byte), args ...string) error {
	n.mu.RLock()
	repoPath := n.repoPath
	n.mu.RUnlock()

	if repoPath == "" {
		return fmt.Errorf("no repository open: call Open() first")
	}

	cmd := exec.CommandContext(ctx, n.gitPath, args...)
	cmd.Dir = repoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start git %v: %w", args, err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		handler(scanner.Bytes())
	}

	if err := cmd.Wait(); err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return &GitError{
			Stderr:   stderr.String(),
			Args:     args,
			ExitCode: exitCode,
		}
	}

	return nil
}

// --- Status ---

func (n *NativeExec) Status(ctx context.Context) (*models.Status, error) {
	data, err := n.run(ctx, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return nil, err
	}

	bi := ParseBranchInfo(data)
	return ParsePorcelainV2(data, bi), nil
}

// --- Log ---

func (n *NativeExec) Log(ctx context.Context, opts LogOptions) ([]*models.Commit, error) {
	args := []string{"log", "--format=" + logFormat}

	if opts.Count > 0 {
		args = append(args, fmt.Sprintf("-%d", opts.Count))
	}
	if opts.All {
		args = append(args, "--all")
	}
	if opts.Merges {
		args = append(args, "--merges")
	}
	if opts.NoMerges {
		args = append(args, "--no-merges")
	}
	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}
	if opts.Since != "" {
		args = append(args, "--since="+opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until="+opts.Until)
	}
	if opts.File != "" {
		args = append(args, "--", opts.File)
	} else if opts.Branch != "" {
		args = append(args, opts.Branch)
	}

	data, err := n.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return ParseLog(data)
}

// --- Diff ---

func (n *NativeExec) Diff(ctx context.Context, opts DiffOptions) (*models.Diff, error) {
	files, err := n.DiffFiles(ctx, opts)
	if err != nil {
		return nil, err
	}

	diff := &models.Diff{Files: files}
	for _, f := range files {
		diff.TotalAdds += f.Additions
		diff.TotalDeletes += f.Deletions
	}
	return diff, nil
}

func (n *NativeExec) DiffFiles(ctx context.Context, opts DiffOptions) ([]models.FileChange, error) {
	// Build common args for both numstat and unified diff
	commonArgs := []string{"diff"}
	if opts.Cached {
		commonArgs = append(commonArgs, "--cached")
	}
	contextLines := 3
	if opts.Context > 0 {
		contextLines = opts.Context
	}
	if opts.A != "" {
		commonArgs = append(commonArgs, opts.A)
	}
	if opts.B != "" {
		commonArgs = append(commonArgs, opts.B)
	}
	pathspec := opts.Pathspec

	// Get numstat for adds/deletes
	numstatArgs := append(commonArgs, "--numstat")
	if pathspec != "" {
		numstatArgs = append(numstatArgs, "--", pathspec)
	}
	data, err := n.run(ctx, numstatArgs...)
	if err != nil {
		return nil, err
	}
	files, err := ParseDiffNumstat(data)
	if err != nil {
		return nil, err
	}

	// If unified diff requested, get the full patch text
	if opts.Unified {
		unifiedArgs := append(commonArgs, fmt.Sprintf("-U%d", contextLines))
		if pathspec != "" {
			unifiedArgs = append(unifiedArgs, "--", pathspec)
		}
		unifiedData, err := n.run(ctx, unifiedArgs...)
		if err != nil {
			return files, nil // return partial; non-fatal
		}
		unifiedLines := strings.Split(string(unifiedData), "\n")

		// Assign unified diff text to each file in order
		diffBlocks := splitUnifiedDiffByFile(unifiedLines)
		for i := range files {
			if i < len(diffBlocks) {
				files[i].UnifiedDiff = strings.Join(diffBlocks[i], "\n")
			}
		}
	}

	return files, nil
}

// splitUnifiedDiffByFile splits unified diff output into per-file blocks.
func splitUnifiedDiffByFile(lines []string) [][]string {
	var blocks [][]string
	var current []string
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") && len(current) > 0 {
			blocks = append(blocks, current)
			current = nil
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

func (n *NativeExec) Show(ctx context.Context, ref string) (*models.Diff, error) {
	args := []string{"show", "--numstat", ref}
	data, err := n.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	files, err := ParseDiffNumstat(data)
	if err != nil {
		return nil, err
	}
	diff := &models.Diff{Files: files}
	for _, f := range files {
		diff.TotalAdds += f.Additions
		diff.TotalDeletes += f.Deletions
	}
	return diff, nil
}

// --- Branches ---

func (n *NativeExec) Branches(ctx context.Context) ([]*models.Branch, error) {
	data, err := n.run(ctx, "branch", "--format="+branchFormat)
	if err != nil {
		return nil, err
	}
	return ParseBranches(data)
}

func (n *NativeExec) CurrentBranch(ctx context.Context) (string, error) {
	data, err := n.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (n *NativeExec) BranchCreate(ctx context.Context, name string) error {
	_, err := n.run(ctx, "branch", name)
	return err
}

func (n *NativeExec) BranchDelete(ctx context.Context, name string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, name)
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) BranchRename(ctx context.Context, oldName, newName string) error {
	_, err := n.run(ctx, "branch", "-m", oldName, newName)
	return err
}

// --- Checkout ---

func (n *NativeExec) Checkout(ctx context.Context, ref string) error {
	_, err := n.run(ctx, "checkout", ref)
	return err
}

func (n *NativeExec) CreateBranchAndCheckout(ctx context.Context, name string) error {
	_, err := n.run(ctx, "checkout", "-b", name)
	return err
}

// --- Merge ---

func (n *NativeExec) Merge(ctx context.Context, branch string) (string, error) {
	out, err := n.run(ctx, "merge", "--no-ff", "--no-stat", branch)
	if err != nil {
		// Try without --no-ff in case of failure (fast-forward-only repos)
		out2, err2 := n.run(ctx, "merge", "--no-stat", branch)
		if err2 != nil {
			return "", fmt.Errorf("merge failed: %w\n%s", err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out2)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

// --- Staging ---

func (n *NativeExec) Add(ctx context.Context, opts AddOptions, files ...string) error {
	args := []string{"add"}
	if opts.IntentToAdd {
		args = append(args, "-N")
	}
	if opts.Force {
		args = append(args, "-f")
	}
	if opts.All {
		args = append(args, "-A")
	}
	if opts.Update {
		args = append(args, "-u")
	}
	if len(files) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, files...)
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) Reset(ctx context.Context, files ...string) error {
	args := []string{"reset"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) Restore(ctx context.Context, files ...string) error {
	args := []string{"restore"}
	args = append(args, files...)
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) RestoreStaged(ctx context.Context, files ...string) error {
	args := []string{"restore", "--staged"}
	args = append(args, files...)
	_, err := n.run(ctx, args...)
	return err
}

// --- Commit ---

func (n *NativeExec) Commit(ctx context.Context, opts CommitOptions) error {
	args := []string{"commit"}
	if opts.AllowEmpty {
		args = append(args, "--allow-empty")
	}
	if opts.Amend {
		args = append(args, "--amend")
	}
	if opts.Signoff {
		args = append(args, "-s")
	}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
		if opts.Body != "" {
			args = append(args, "-m", opts.Body)
		}
	}
	_, err := n.run(ctx, args...)
	return err
}

// --- Remotes ---

func (n *NativeExec) RemoteList(ctx context.Context) ([]*models.Remote, error) {
	data, err := n.run(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}
	return ParseRemotes(data)
}

// ApplyPatch applies a unified diff patch to the working tree or index.
// If cached is true, it applies --cached (stages the patch).
func (n *NativeExec) ApplyPatch(ctx context.Context, patch string, cached bool) error {
	n.mu.RLock()
	repoPath := n.repoPath
	n.mu.RUnlock()

	if repoPath == "" {
		return fmt.Errorf("no repository open: call Open() first")
	}

	args := []string{"apply", "--unidiff-zero"}
	if cached {
		args = append(args, "--cached")
	}
	cmd := exec.CommandContext(ctx, n.gitPath, args...)
	cmd.Dir = repoPath
	cmd.Stdin = strings.NewReader(patch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply patch: %w\n%s", err, string(out))
	}
	return nil
}

func (n *NativeExec) RemoteAdd(ctx context.Context, name, url string) error {
	_, err := n.run(ctx, "remote", "add", name, url)
	return err
}

func (n *NativeExec) RemoteRemove(ctx context.Context, name string) error {
	_, err := n.run(ctx, "remote", "remove", name)
	return err
}

// --- Sync ---

func (n *NativeExec) Fetch(ctx context.Context, remote string, prune bool) error {
	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
	}
	if prune {
		args = append(args, "--prune")
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) Pull(ctx context.Context, opts PullOptions) error {
	args := []string{"pull"}
	if opts.Rebase {
		args = append(args, "--rebase")
	}
	if opts.FFOnly {
		args = append(args, "--ff-only")
	}
	if opts.Remote != "" {
		args = append(args, opts.Remote)
	}
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) Push(ctx context.Context, opts PushOptions) error {
	args := []string{"push"}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}
	if opts.SetUpstream {
		args = append(args, "-u")
	}
	if opts.Remote != "" {
		args = append(args, opts.Remote)
	}
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}
	_, err := n.run(ctx, args...)
	return err
}

// --- Stash ---

func (n *NativeExec) StashList(ctx context.Context) ([]*models.Stash, error) {
	data, err := n.run(ctx, "stash", "list", "--format=%H%x09%gs")
	if err != nil {
		// No stashes → exit 0 + empty output, not an error.
		// Real errors (bad repo, permissions) produce non-zero exit.
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	return ParseStashList(data)
}

func (n *NativeExec) StashPush(ctx context.Context, opts StashOptions, message string) error {
	args := []string{"stash", "push"}
	if opts.KeepIndex {
		args = append(args, "--keep-index")
	}
	if opts.IncludeUntracked {
		args = append(args, "--include-untracked")
	}
	if opts.Staged {
		args = append(args, "--staged")
	}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) StashPop(ctx context.Context, index int) error {
	args := []string{"stash", "pop"}
	if index >= 0 {
		args = append(args, fmt.Sprintf("stash@{%d}", index))
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) StashApply(ctx context.Context, index int) error {
	args := []string{"stash", "apply"}
	if index >= 0 {
		args = append(args, fmt.Sprintf("stash@{%d}", index))
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) StashDrop(ctx context.Context, index int) error {
	args := []string{"stash", "drop"}
	if index >= 0 {
		args = append(args, fmt.Sprintf("stash@{%d}", index))
	}
	_, err := n.run(ctx, args...)
	return err
}

// --- Misc ---

func (n *NativeExec) MergeBase(ctx context.Context, a, b string) (string, error) {
	data, err := n.run(ctx, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (n *NativeExec) RefExists(ctx context.Context, ref string) (bool, error) {
	_, err := n.run(ctx, "rev-parse", "--verify", ref)
	if err != nil {
		var gitErr *GitError
		if asGitErr(err, &gitErr) && gitErr.ExitCode == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (n *NativeExec) RevParse(ctx context.Context, ref string) (string, error) {
	data, err := n.run(ctx, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (n *NativeExec) IsWorkTreeClean(ctx context.Context) (bool, error) {
	data, err := n.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(data)) == 0, nil
}

// RevList returns commit hashes reachable from ref, up to count.
func (n *NativeExec) RevList(ctx context.Context, ref string, count int) ([]string, error) {
	args := []string{"rev-list"}
	if count > 0 {
		args = append(args, fmt.Sprintf("-%d", count))
	}
	args = append(args, ref)
	data, err := n.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return ParseRevList(data)
}

// asGitErr unwraps err to *GitError if possible.
func asGitErr(err error, target **GitError) bool {
	if err == nil {
		return false
	}
	*target, _ = err.(*GitError) // direct cast OK here — we wrap at origin
	return *target != nil
}
