package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

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
	s := ParsePorcelainV2(data, bi)

	// Detect merge/rebase/cherry-pick/revert state by checking for sentinel files.
	gitDir := n.repoPath + "/.git"
	if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
		s.IsMerging = fileExists(gitDir + "/MERGE_HEAD")
		s.IsRebasing = fileExists(gitDir + "/REBASE_HEAD")
		s.IsCherryPick = fileExists(gitDir + "/CHERRY_PICK_HEAD")
		s.IsReverting = fileExists(gitDir + "/REVERT_HEAD")
	}
	// Also detect via porcelain v2 unmerged entries.
	// If any file has StatusUpdatedButUnmerged, we are in a merge/conflict state.
	for _, f := range s.Files {
		if f.Staged == models.StatusUpdatedButUnmerged || f.Unstaged == models.StatusUpdatedButUnmerged {
			s.IsMerging = true
			break
		}
	}

	return s, nil
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

// gitDir returns the path to the .git directory for the current repository.
func (n *NativeExec) gitDir(ctx context.Context) string {
	// Try the common case first.
	gitDir := n.repoPath + "/.git"
	if fi, err := os.Stat(gitDir); err == nil {
		if fi.IsDir() {
			return gitDir
		}
		// It might be a gitfile (worktree, submodule).
		data, err := os.ReadFile(gitDir)
		if err == nil {
			line := strings.TrimSpace(string(data))
			if strings.HasPrefix(line, "gitdir: ") {
				return strings.TrimPrefix(line, "gitdir: ")
			}
		}
	}
	// Fallback: ask git.
	data, err := n.run(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return n.repoPath + "/.git"
	}
	return strings.TrimSpace(string(data))
}

// --- Log ---

func (n *NativeExec) Log(ctx context.Context, opts LogOptions) ([]*models.Commit, error) {
	args := []string{"log", "--format=" + logFormat}

	if opts.Graph {
		args = append(args, "--topo-order")
	}
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
		// Empty repo / unborn HEAD: git log exits with code 128.
		var gitErr *GitError
		if asGitErr(err, &gitErr) && gitErr.ExitCode == 128 {
			return []*models.Commit{}, nil
		}
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
		// Empty repo / unborn HEAD: git branch exits with code 128.
		// Return empty slice instead of propagating the error.
		var gitErr *GitError
		if asGitErr(err, &gitErr) && gitErr.ExitCode == 128 {
			return []*models.Branch{}, nil
		}
		return nil, err
	}
	return ParseBranches(data)
}

func (n *NativeExec) CurrentBranch(ctx context.Context) (string, error) {
	data, err := n.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		// Unborn HEAD (empty repo): rev-parse exits with 128.
		// Return "HEAD" instead of propagating the error.
		var gitErr *GitError
		if asGitErr(err, &gitErr) && gitErr.ExitCode == 128 {
			return "HEAD", nil
		}
		return "", err
	}
	branch := strings.TrimSpace(string(data))
	// When detached HEAD, rev-parse prints "HEAD"
	if branch == "HEAD" {
		return branch, nil
	}
	return branch, nil
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

// --- Rebase ---

func (n *NativeExec) RebaseSequence(ctx context.Context, opts models.RebaseSequenceOptions) (*models.RebaseResult, error) {
	// Generate the rebase todo list
	var todoLines []string
	for _, commit := range opts.Commits {
		line := string(commit.Action) + " " + commit.SHA
		if commit.Action == models.RebaseReword && commit.NewMessage != "" {
			line += " # reword " + commit.NewMessage
		}
		todoLines = append(todoLines, line)
	}

	todoContent := strings.Join(todoLines, "\n") + "\n"

	// Write todo to a temp file
	tmpFile, err := os.CreateTemp("", "git-rebase-todo-*")
	if err != nil {
		return nil, fmt.Errorf("create temp todo: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(todoContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("write todo: %w", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Execute rebase with GIT_SEQUENCE_EDITOR pointing to our todo
	n.mu.RLock()
	repoPath := n.repoPath
	n.mu.RUnlock()

	// We need a wrapper script because GIT_SEQUENCE_EDITOR receives the
	// target filename as an argument: "$EDITOR <file>"
	// We want to copy our todo over it. Create a shell script.
	editorCmd := fmt.Sprintf("cp %s \"$1\"", tmpPath)

	cmd := exec.CommandContext(ctx, n.gitPath, "rebase", "-i", opts.Onto)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(),
		"GIT_SEQUENCE_EDITOR="+editorCmd,
		"GIT_EDITOR="+editorCmd,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Rebase may have conflicts
		return &models.RebaseResult{
			Success: false,
			Message: strings.TrimSpace(string(output)),
		}, nil
	}

	return &models.RebaseResult{
		Success: true,
		Message: strings.TrimSpace(string(output)),
	}, nil
}

// --- Commit operations ---

func (n *NativeExec) CherryPick(ctx context.Context, sha string) error {
	_, err := n.run(ctx, "cherry-pick", sha)
	return err
}

// CherryPickNoCommit stages changes from a commit without creating a commit.
func (n *NativeExec) CherryPickNoCommit(ctx context.Context, sha string) error {
	_, err := n.run(ctx, "cherry-pick", "--no-commit", sha)
	return err
}

func (n *NativeExec) Revert(ctx context.Context, sha string) error {
	_, err := n.run(ctx, "revert", "--no-edit", sha)
	return err
}

func (n *NativeExec) ResetCommit(ctx context.Context, sha, mode string) error {
	flag := "--" + mode // --soft, --mixed, --hard
	_, err := n.run(ctx, "reset", flag, sha)
	return err
}

// --- Tags ---

func (n *NativeExec) TagList(ctx context.Context) ([]string, error) {
	data, err := n.run(ctx, "tag", "-l", "--sort=-creatordate")
	if err != nil {
		return nil, err
	}
	out := strings.TrimSpace(string(data))
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
}

func (n *NativeExec) TagCreate(ctx context.Context, name, target, message string) error {
	args := []string{"tag"}
	if message != "" {
		args = append(args, "-a", name, "-m", message)
	} else {
		args = append(args, name)
	}
	if target != "" {
		args = append(args, target)
	}
	_, err := n.run(ctx, args...)
	return err
}

func (n *NativeExec) TagDelete(ctx context.Context, name string) error {
	_, err := n.run(ctx, "tag", "-d", name)
	return err
}

// --- Config ---

func (n *NativeExec) ConfigGet(ctx context.Context, key string) (string, error) {
	data, err := n.run(ctx, "config", key)
	if err != nil {
		return "", nil // key may not exist
	}
	return strings.TrimSpace(string(data)), nil
}

func (n *NativeExec) ConfigSet(ctx context.Context, key, value string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)
	_, err := n.run(ctx, args...)
	return err
}

// --- Conflict resolution ---

func (n *NativeExec) CheckoutOurs(ctx context.Context, file string) error {
	_, err := n.run(ctx, "checkout", "--ours", "--", file)
	if err != nil {
		return err
	}
	// Stage the resolved file
	_, err = n.run(ctx, "add", file)
	return err
}

func (n *NativeExec) CheckoutTheirs(ctx context.Context, file string) error {
	_, err := n.run(ctx, "checkout", "--theirs", "--", file)
	if err != nil {
		return err
	}
	_, err = n.run(ctx, "add", file)
	return err
}

// ConflictFiles returns a list of files with unresolved merge conflicts.
func (n *NativeExec) ConflictFiles(ctx context.Context) ([]models.ConflictFile, error) {
	data, err := n.run(ctx, "ls-files", "-u")
	if err != nil {
		return nil, err
	}

	// git ls-files -u output:
	// 100644 123abc 1	path/to/file
	// 100644 456def 2	path/to/file
	// 100644 789ghi 3	path/to/file
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	fileMap := make(map[string]*models.ConflictFile)
	var order []string

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		sha := parts[1]
		stage := parts[2]
		path := parts[3]

		if _, ok := fileMap[path]; !ok {
			fileMap[path] = &models.ConflictFile{Path: path}
			order = append(order, path)
		}

		switch stage {
		case "1":
			fileMap[path].AncestorSHA = sha
		case "2":
			fileMap[path].OursSHA = sha
		case "3":
			fileMap[path].TheirsSHA = sha
		}
	}

	result := make([]models.ConflictFile, 0, len(order))
	for _, path := range order {
		cf := fileMap[path]
		// Count conflict blocks by reading the working tree file
		cf.BlockCount = countConflictBlocks(n.repoPath + "/" + path)
		result = append(result, *cf)
	}
	return result, nil
}

// countConflictBlocks reads a file and counts the number of conflict regions (<<<<<<<).
func countConflictBlocks(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "<<<<<<< ") {
			count++
		}
	}
	return count
}

// GetMergeConflictDetail returns detailed 3-way conflict info for a single file.
func (n *NativeExec) GetMergeConflictDetail(ctx context.Context, file string) (*models.MergeConflictDetail, error) {
	detail := &models.MergeConflictDetail{Path: file}

	// Get ancestor (stage 1) — may not exist for add/add conflicts
	ancestorData, _ := n.run(ctx, "show", ":1:"+file)
	if len(ancestorData) > 0 {
		detail.Ancestor = string(ancestorData)
	}

	// Get ours (stage 2)
	oursData, err := n.run(ctx, "show", ":2:"+file)
	if err != nil {
		return nil, fmt.Errorf("get ours (:2:%s): %w", file, err)
	}
	detail.Ours = string(oursData)

	// Get theirs (stage 3)
	theirsData, err := n.run(ctx, "show", ":3:"+file)
	if err != nil {
		return nil, fmt.Errorf("get theirs (:3:%s): %w", file, err)
	}
	detail.Theirs = string(theirsData)

	// Read the working tree file to parse conflict markers
	wtPath := n.repoPath + "/" + file
	wtData, err := os.ReadFile(wtPath)
	if err != nil {
		return nil, fmt.Errorf("read working tree file %s: %w", file, err)
	}
	wtContent := string(wtData)
	detail.RawContent = wtContent

	// Parse conflict markers
	blocks, hasMerge := parseConflictMarkers(wtContent)
	detail.Blocks = blocks
	detail.HasMerge = hasMerge

	return detail, nil
}

// parseConflictMarkers parses conflict markers from file content.
// <<<<<<< ours-ref
// ours content lines
// =======
// theirs content lines
// >>>>>>> theirs-ref
func parseConflictMarkers(content string) ([]models.ConflictBlock, bool) {
	var blocks []models.ConflictBlock
	lines := strings.Split(content, "\n")
	blockIndex := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "<<<<<<< ") {
			block := models.ConflictBlock{
				Index: blockIndex,
				State: "unresolved",
			}
			// Ours start line
			block.OursStart = i + 1
			i++ // move past <<<<<<<
			var oursLines []string
			for i < len(lines) && !strings.HasPrefix(lines[i], "=======") {
				oursLines = append(oursLines, lines[i])
				i++
			}
			block.OursEnd = i
			block.Ours = strings.Join(oursLines, "\n")

			if i < len(lines) && strings.HasPrefix(lines[i], "=======") {
				i++ // move past =======
			}

			block.TheirsStart = i + 1
			var theirLines []string
			for i < len(lines) && !strings.HasPrefix(lines[i], ">>>>>>> ") {
				theirLines = append(theirLines, lines[i])
				i++
			}
			block.TheirsEnd = i
			block.Theirs = strings.Join(theirLines, "\n")

			// Default resolution: use ours initially as placeholder
			block.Resolved = block.Ours
			blockIndex++

			blocks = append(blocks, block)
		}
	}

	return blocks, len(blocks) > 0
}

// StageResolvedFile writes resolved content to the working tree and stages it.
func (n *NativeExec) StageResolvedFile(ctx context.Context, file, content string) error {
	wtPath := n.repoPath + "/" + file
	if err := os.WriteFile(wtPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write resolved file %s: %w", file, err)
	}
	_, err := n.run(ctx, "add", file)
	return err
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

func (n *NativeExec) RemoteRename(ctx context.Context, oldName, newName string) error {
	_, err := n.run(ctx, "remote", "rename", oldName, newName)
	return err
}

func (n *NativeExec) RemoteSetURL(ctx context.Context, name, url string) error {
	_, err := n.run(ctx, "remote", "set-url", name, url)
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

// --- Reflog ---

func (n *NativeExec) Reflog(ctx context.Context, count int) ([]models.ReflogEntry, error) {
	// Format: %gd = reflog selector (HEAD@{N}), %H = hash, %gs = reflog subject, %ci = committer date (ISO)
	format := "%gd|%H|%gs|%ci"
	args := []string{"reflog", fmt.Sprintf("--format=%s", format)}
	if count > 0 {
		args = append(args, fmt.Sprintf("-%d", count))
	}
	data, err := n.run(ctx, args...)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(bytes.TrimSpace(data)), "\n")
	entries := make([]models.ReflogEntry, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		entry := parseReflogLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	return entries, nil
}

// parseReflogLine parses a single reflog line in format: %gd|%H|%gs|%ci
func parseReflogLine(line string) *models.ReflogEntry {
	parts := strings.SplitN(line, "|", 4)
	if len(parts) < 4 {
		return nil
	}

	// Extract sequence number from selector like "HEAD@{0}"
	seq := 0
	selector := parts[0]
	if n, err := fmt.Sscanf(selector, "HEAD@{%d}", &seq); err == nil && n == 1 {
		// ok
	}

	hash := parts[1]
	subject := parts[2]

	ts, err := time.Parse("2006-01-02 15:04:05 -0700", parts[3])
	if err != nil {
		// Try without timezone offset
		ts, err = time.Parse("2006-01-02 15:04:05", parts[3])
		if err != nil {
			ts = time.Now()
		}
	}

	entry := &models.ReflogEntry{
		Sequence:  seq,
		Hash:      hash,
		Subject:   subject,
		Timestamp: ts,
		Action:    categorizeReflogAction(subject),
	}
	entry.Undoable = models.UndoableActions[entry.Action]
	return entry
}

// categorizeReflogAction categorises a reflog subject line into an action type.
func categorizeReflogAction(subject string) models.ReflogAction {
	switch {
	case strings.HasPrefix(subject, "commit"):
		return models.ReflogCommit
	case strings.HasPrefix(subject, "reset"):
		return models.ReflogReset
	case strings.HasPrefix(subject, "checkout"):
		return models.ReflogCheckout
	case strings.HasPrefix(subject, "merge"):
		return models.ReflogMerge
	case strings.HasPrefix(subject, "rebase"):
		return models.ReflogRebase
	case strings.HasPrefix(subject, "cherry-pick"):
		return models.ReflogCherryPick
	case strings.HasPrefix(subject, "revert"):
		return models.ReflogRevert
	case strings.HasPrefix(subject, "Branch"):
		return models.ReflogBranch
	case strings.HasPrefix(subject, "commit (amend)"):
		return models.ReflogAmend
	case strings.HasPrefix(subject, "commit (cherry-pick"):
		return models.ReflogCherryPick
	default:
		return models.ReflogUnknown
	}
}

func (n *NativeExec) UndoLastAction(ctx context.Context) (string, error) {
	// Get the last 2 reflog entries: HEAD@{0} = current, HEAD@{1} = before
	entries, err := n.Reflog(ctx, 2)
	if err != nil {
		return "", fmt.Errorf("read reflog: %w", err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no reflog entries to undo")
	}

	last := entries[0]
	if !last.Undoable {
		return "", fmt.Errorf("last action (%s) cannot be automatically undone", last.Subject)
	}

	// For some actions we have ORIG_HEAD, for others we reset to HEAD@{1}.
	// Strategy per action type:
	var description string

	switch last.Action {
	case models.ReflogCommit:
		// Soft reset: undo commit, keep changes staged
		_, err = n.run(ctx, "reset", "--soft", "HEAD~1")
		description = fmt.Sprintf("Undid commit: %s", truncateReflogSubject(last.Subject))

	case models.ReflogReset, models.ReflogMerge, models.ReflogRevert:
		// These set ORIG_HEAD — reset back to it, keep working tree
		_, err = n.run(ctx, "reset", "--merge", "ORIG_HEAD")
		description = fmt.Sprintf("Undid %s: %s", last.Action, truncateReflogSubject(last.Subject))

	case models.ReflogCherryPick:
		// If still in cherry-pick, abort. Otherwise reset to ORIG_HEAD.
		_, errCherry := n.run(ctx, "cherry-pick", "--abort")
		if errCherry != nil {
			// Not in progress, try ORIG_HEAD
			_, err = n.run(ctx, "reset", "--keep", "ORIG_HEAD")
		} else {
			err = nil
		}
		description = fmt.Sprintf("Undid cherry-pick: %s", truncateReflogSubject(last.Subject))

	case models.ReflogBranch:
		// Branch create/delete. Check if it was a delete.
		if strings.Contains(last.Subject, "delete") || strings.Contains(last.Subject, "deleted") {
			// Branch deletion — check if we have a previous hash in reflog
			if len(entries) > 1 {
				// The subject tells us which branch. Parse it out.
				branchName := parseBranchNameFromSubject(last.Subject)
				if branchName != "" {
					_, err = n.run(ctx, "branch", branchName, entries[1].Hash)
					description = fmt.Sprintf("Restored branch %s at %s", branchName, entries[1].Hash[:8])
				} else {
					err = fmt.Errorf("cannot determine deleted branch name")
				}
			} else {
				err = fmt.Errorf("insufficient reflog entries to restore branch")
			}
		} else {
			// Branch creation — delete it
			branchName := parseBranchNameFromSubject(last.Subject)
			if branchName != "" {
				_, err = n.run(ctx, "branch", "-d", branchName)
				description = fmt.Sprintf("Removed branch %s", branchName)
			} else {
				err = fmt.Errorf("cannot determine created branch name")
			}
		}

	case models.ReflogAmend:
		// Undo amend: reset to ORIG_HEAD
		_, err = n.run(ctx, "reset", "--soft", "ORIG_HEAD")
		description = fmt.Sprintf("Undid amend: %s", truncateReflogSubject(last.Subject))

	default:
		// Generic: soft reset to previous reflog state (safe — preserves working tree)
		if len(entries) > 1 {
			_, err = n.run(ctx, "reset", "--soft", entries[1].Hash)
			description = fmt.Sprintf("Undid %s", truncateReflogSubject(last.Subject))
		} else {
			err = fmt.Errorf("insufficient reflog history to undo")
		}
	}

	if err != nil {
		return "", err
	}
	return description, nil
}

// truncateReflogSubject truncates a reflog subject for display.
func truncateReflogSubject(subject string) string {
	if len(subject) > 60 {
		return subject[:57] + "..."
	}
	return subject
}

// parseBranchNameFromSubject extracts a branch name from a reflog subject line.
// Examples:
//
//	"Branch: renamed refs/heads/old to refs/heads/new"
//	"Branch: deleted refs/heads/feature"
//	"checkout: moving from main to feature"
func parseBranchNameFromSubject(subject string) string {
	// "Branch: deleted refs/heads/<name>"
	// "Branch: created refs/heads/<name>"
	if strings.Contains(subject, "refs/heads/") {
		idx := strings.Index(subject, "refs/heads/")
		rest := subject[idx+len("refs/heads/"):]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
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
