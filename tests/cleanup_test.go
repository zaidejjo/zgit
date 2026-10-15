package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initCleanupRepo builds: main + feature-a (merged) + feature-b (unmerged)
// + dev (merged, protected). Returns dir.
func initCleanupRepo(t *testing.T) string {
	t.Helper()
	gitTestPath(t)
	dir := t.TempDir()
	run := func(args ...string) {
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
	run("init", "-q", "-b", "main")
	run("config", "user.email", "t@t.co")
	run("config", "user.name", "t")
	commit := func(file, content, msg string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		run("add", ".")
		run("commit", "-qm", msg)
	}
	commit("base.txt", "base\n", "base")
	run("checkout", "-qb", "feature-a")
	commit("a.txt", "a\n", "work a")
	run("checkout", "-q", "main")
	run("merge", "-q", "--no-ff", "feature-a", "-m", "merge a")
	run("checkout", "-qb", "feature-b")
	commit("b.txt", "b\n", "work b")
	run("checkout", "-q", "main")
	run("checkout", "-qb", "dev")
	commit("d.txt", "d\n", "work d")
	run("checkout", "-q", "main")
	run("merge", "-q", "--no-ff", "dev", "-m", "merge d")
	return dir
}

func TestCleanupScan(t *testing.T) {
	dir := initCleanupRepo(t)
	n, ctx := openTestRepo(t, dir)

	merged, err := n.BranchMerged(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	has := func(name string) bool {
		for _, b := range merged {
			if b == name {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"main", "feature-a", "dev"} {
		if !has(want) {
			t.Errorf("merged should contain %s: %v", want, merged)
		}
	}
	if has("feature-b") {
		t.Errorf("merged should not contain unmerged feature-b: %v", merged)
	}
	if _, err := n.BranchMerged(ctx, ""); err == nil {
		t.Errorf("empty base should fail")
	}
}

func TestEngineCleanupScan(t *testing.T) {
	dir := initCleanupRepo(t)
	e := testEngine(t)

	candidates, withheld, err := e.CleanupScan(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Branch != "feature-a" {
		t.Fatalf("candidates = %+v, want [feature-a]", candidates)
	}
	if candidates[0].LastMessage == "" {
		t.Errorf("candidate should carry last commit message")
	}
	reasons := map[string]bool{}
	for _, w := range withheld {
		reasons[w.Branch] = true
	}
	for _, want := range []string{"main", "dev"} {
		if !reasons[want] {
			t.Errorf("withheld should contain %s: %+v", want, withheld)
		}
	}

	// Default base resolves to current branch (main here).
	c2, _, err := e.CleanupScan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(c2) != 1 {
		t.Errorf("default-base candidates = %+v", c2)
	}

	// Safe delete works for merged, refuses unmerged.
	if err := e.DeleteRepoBranch(dir, "feature-a"); err != nil {
		t.Errorf("delete merged: %v", err)
	}
	if err := e.DeleteRepoBranch(dir, "feature-b"); err == nil {
		t.Errorf("delete unmerged should fail with safe -d")
	}
}

func TestProjectCleanupScan(t *testing.T) {
	dir := initCleanupRepo(t)
	e := testEngine(t)
	id, err := e.CreateProject("clean", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.AddProjectRepo(id, dir); err != nil {
		t.Fatal(err)
	}
	candidates, _, err := e.ProjectCleanupScan(id, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].RepoPath != dir {
		t.Fatalf("project candidates = %+v", candidates)
	}
}
