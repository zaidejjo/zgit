package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/git"
)

func TestPRRefspec(t *testing.T) {
	refspec, branch := git.PRRefspec(42)
	if branch != "pr/42" {
		t.Errorf("branch = %q, want pr/42", branch)
	}
	if refspec != "+pull/42/head:refs/heads/pr/42" {
		t.Errorf("refspec = %q", refspec)
	}
}

// initRemotePair creates a work repo with a bare repo as its origin.
func initRemotePair(t *testing.T) (work string) {
	t.Helper()
	gitTestPath(t)
	work = t.TempDir()
	bare := filepath.Join(t.TempDir(), "origin.git")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t.co",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t.co",
			"GIT_CONFIG_NOSYSTEM=1",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("", "init", "--bare", "-q", bare)
	run(work, "init", "-q", "-b", "main")
	run(work, "config", "user.email", "t@t.co")
	run(work, "config", "user.name", "t")
	run(work, "remote", "add", "origin", bare)
	if err := os.WriteFile(filepath.Join(work, "f.txt"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(work, "add", ".")
	run(work, "commit", "-qm", "init")
	run(work, "push", "-q", "origin", "main")
	return work
}

func TestFetchRefspec(t *testing.T) {
	work := initRemotePair(t)
	n, ctx := openTestRepo(t, work)

	// Simulate `gh pr checkout` mechanics: fetch a remote branch
	// into a pr/N-style local branch, then check it out.
	if err := n.FetchRefspec(ctx, "origin", "main:pr/7"); err != nil {
		t.Fatalf("FetchRefspec: %v", err)
	}
	exists, err := n.RefExists(ctx, "refs/heads/pr/7")
	if err != nil || !exists {
		t.Fatalf("refs/heads/pr/7 exists=%v err=%v", exists, err)
	}
	if err := n.Checkout(ctx, "pr/7"); err != nil {
		t.Fatalf("Checkout pr/7: %v", err)
	}
	cur, err := n.CurrentBranch(ctx)
	if err != nil || cur != "pr/7" {
		t.Errorf("current branch = %q, want pr/7", cur)
	}

	if err := n.FetchRefspec(ctx, "", "main:pr/7"); err == nil {
		t.Errorf("FetchRefspec(empty remote) = nil, want error")
	}
	if err := n.FetchRefspec(ctx, "origin", "   "); err == nil {
		t.Errorf("FetchRefspec(empty refspec) = nil, want error")
	}
}
