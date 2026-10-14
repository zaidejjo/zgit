// Package integration holds end-to-end tests that exercise the public
// core API against real git repositories in temp dirs.
// Kept separate from unit tests next to the source so `go test ./...`
// runs both, while integration coverage stays in one place.
package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// gitTestPath returns the git binary or skips the test when git is unavailable.
func gitTestPath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not found in PATH")
	}
	return p
}

// initTestRepo creates a committed git repo in a temp dir and returns its path.
func initTestRepo(t *testing.T, gitBin string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitBin, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t.co",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t.co",
			"GIT_CONFIG_NOSYSTEM=1", "HOME="+dir,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "t@t.co")
	run("config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-qm", "init")
	return dir
}

func openTestRepo(t *testing.T, path string) (*git.NativeExec, context.Context) {
	t.Helper()
	n, err := git.NewNativeExec("")
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Open(path); err != nil {
		t.Fatal(err)
	}
	return n, context.Background()
}

func TestWorktreeAddListRemove(t *testing.T) {
	gitBin := gitTestPath(t)
	repo := initTestRepo(t, gitBin)
	n, ctx := openTestRepo(t, repo)

	linked := filepath.Join(t.TempDir(), "linked")
	if err := n.WorktreeAdd(ctx, linked, "feature/x", models.WorktreeAddOptions{CreateBranch: true}); err != nil {
		t.Fatalf("WorktreeAdd: %v", err)
	}

	list, err := n.WorktreeList(ctx)
	if err != nil {
		t.Fatalf("WorktreeList: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	if !list[0].IsMain {
		t.Errorf("list[0].IsMain = false, want true (main first)")
	}
	if list[1].Branch != "feature/x" {
		t.Errorf("list[1].Branch = %q, want %q", list[1].Branch, "feature/x")
	}
	if list[1].Head == "" {
		t.Errorf("list[1].Head is empty")
	}

	if err := n.WorktreeRemove(ctx, linked, false); err != nil {
		t.Fatalf("WorktreeRemove: %v", err)
	}
	list, err = n.WorktreeList(ctx)
	if err != nil {
		t.Fatalf("WorktreeList after remove: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1 after remove", len(list))
	}
}

func TestWorktreeLockUnlock(t *testing.T) {
	gitBin := gitTestPath(t)
	repo := initTestRepo(t, gitBin)
	n, ctx := openTestRepo(t, repo)

	linked := filepath.Join(t.TempDir(), "linked")
	if err := n.WorktreeAdd(ctx, linked, "main", models.WorktreeAddOptions{Force: true}); err != nil {
		t.Fatalf("WorktreeAdd: %v", err)
	}
	if err := n.WorktreeLock(ctx, linked, "testing"); err != nil {
		t.Fatalf("WorktreeLock: %v", err)
	}

	list, err := n.WorktreeList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var found *models.Worktree
	for _, w := range list {
		if w.Path == linked {
			found = w
		}
	}
	if found == nil {
		t.Fatalf("linked worktree %q not in list", linked)
	}
	if !found.Locked || found.LockReason != "testing" {
		t.Errorf("Locked=%v Reason=%q, want true %q", found.Locked, found.LockReason, "testing")
	}

	if err := n.WorktreeUnlock(ctx, linked); err != nil {
		t.Fatalf("WorktreeUnlock: %v", err)
	}
	if err := n.WorktreeRemove(ctx, linked, true); err != nil {
		t.Fatalf("WorktreeRemove after unlock: %v", err)
	}
}

func TestWorktreeRemoveMainRefused(t *testing.T) {
	gitBin := gitTestPath(t)
	repo := initTestRepo(t, gitBin)
	n, ctx := openTestRepo(t, repo)

	if err := n.WorktreeRemove(ctx, repo, true); err == nil {
		t.Errorf("WorktreeRemove(main, force) = nil, want refusal error")
	}
}

func TestWorktreeRemoveMainRefusedRelative(t *testing.T) {
	gitBin := gitTestPath(t)
	repo := initTestRepo(t, gitBin)
	n, ctx := openTestRepo(t, repo)

	// Relative "." must resolve against the repo dir, not the test cwd.
	if err := n.WorktreeRemove(ctx, ".", true); err == nil {
		t.Errorf("WorktreeRemove(repo, %q) = nil, want refusal error", ".")
	}
}

func TestWorktreeAddValidation(t *testing.T) {
	gitBin := gitTestPath(t)
	repo := initTestRepo(t, gitBin)
	n, ctx := openTestRepo(t, repo)

	if err := n.WorktreeAdd(ctx, "   ", "main", models.WorktreeAddOptions{}); err == nil {
		t.Errorf("WorktreeAdd(empty path) = nil, want error")
	}
	if err := n.WorktreeAdd(ctx, filepath.Join(t.TempDir(), "x"), "", models.WorktreeAddOptions{CreateBranch: true}); err == nil {
		t.Errorf("WorktreeAdd(create-branch, empty name) = nil, want error")
	}
}

func TestWorktreePrune(t *testing.T) {
	gitBin := gitTestPath(t)
	repo := initTestRepo(t, gitBin)
	n, ctx := openTestRepo(t, repo)

	parent := t.TempDir()
	linked := filepath.Join(parent, "gone")
	if err := n.WorktreeAdd(ctx, linked, "main", models.WorktreeAddOptions{Force: true}); err != nil {
		t.Fatalf("WorktreeAdd: %v", err)
	}
	// Simulate manual deletion on disk.
	if err := os.RemoveAll(linked); err != nil {
		t.Fatal(err)
	}
	if err := n.WorktreePrune(ctx); err != nil {
		t.Fatalf("WorktreePrune: %v", err)
	}
	list, err := n.WorktreeList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].IsMain {
		t.Errorf("after prune: got %d worktrees, want only main", len(list))
	}
}
