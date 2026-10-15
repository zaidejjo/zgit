package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// runGit executes git with per-call author overrides.
func runGit(t *testing.T, dir, author, email string, args ...string) {
	t.Helper()
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not found in PATH")
	}
	cmd := exec.Command(gitBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+author, "GIT_AUTHOR_EMAIL="+email,
		"GIT_COMMITTER_NAME="+author, "GIT_COMMITTER_EMAIL="+email,
		"GIT_CONFIG_NOSYSTEM=1",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// initBlameRepo creates a repo with two commits on notes.txt by different authors.
func initBlameRepo(t *testing.T) string {
	t.Helper()
	gitTestPath(t)
	dir := t.TempDir()
	init := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	init("init", "-q", "-b", "main")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("line one\nline two\n")
	runGit(t, dir, "Alice", "alice@ex.com", "add", ".")
	runGit(t, dir, "Alice", "alice@ex.com", "commit", "-qm", "add notes")
	write("line one edited\nline two\n")
	runGit(t, dir, "Bob", "bob@ex.com", "commit", "-qam", "edit first line")
	return dir
}

func TestBlameAttributesLines(t *testing.T) {
	repo := initBlameRepo(t)
	n, ctx := openTestRepo(t, repo)

	lines, err := n.Blame(ctx, "notes.txt", models.BlameOptions{})
	if err != nil {
		t.Fatalf("Blame: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("len = %d, want 2", len(lines))
	}
	if lines[0].Author != "Bob" || lines[0].Content != "line one edited" {
		t.Errorf("[0] = %q by %q, want edited line by Bob", lines[0].Content, lines[0].Author)
	}
	if lines[1].Author != "Alice" || lines[1].Content != "line two" {
		t.Errorf("[1] = %q by %q, want untouched line by Alice", lines[1].Content, lines[1].Author)
	}
	if lines[0].Hash == "" || lines[0].Summary == "" {
		t.Errorf("[0] missing hash/summary: %+v", lines[0])
	}
}

func TestBlameLineRange(t *testing.T) {
	repo := initBlameRepo(t)
	n, ctx := openTestRepo(t, repo)

	lines, err := n.Blame(ctx, "notes.txt", models.BlameOptions{Lines: "2,2"})
	if err != nil {
		t.Fatalf("Blame -L: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("len = %d, want 1", len(lines))
	}
	if lines[0].LineNo != 2 || lines[0].Content != "line two" {
		t.Errorf("got %+v, want line 2 %q", lines[0], "line two")
	}
}

func TestBlameValidation(t *testing.T) {
	repo := initBlameRepo(t)
	n, ctx := openTestRepo(t, repo)

	if _, err := n.Blame(ctx, "   ", models.BlameOptions{}); err == nil {
		t.Errorf("Blame(empty path) = nil, want error")
	}
	if _, err := n.Blame(ctx, "notes.txt", models.BlameOptions{Lines: "abc"}); err == nil {
		t.Errorf("Blame(bad range) = nil, want error")
	}
}

func TestLogFollowRename(t *testing.T) {
	gitTestPath(t)
	dir := t.TempDir()
	init := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	init("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "old.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "Alice", "alice@ex.com", "add", ".")
	runGit(t, dir, "Alice", "alice@ex.com", "commit", "-qm", "add old")
	runGit(t, dir, "Alice", "alice@ex.com", "mv", "old.txt", "new.txt")
	runGit(t, dir, "Alice", "alice@ex.com", "commit", "-qm", "rename")

	n, ctx := openTestRepo(t, dir)

	plain, err := n.Log(ctx, git.LogOptions{File: "new.txt"})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	followed, err := n.Log(ctx, git.LogOptions{File: "new.txt", Follow: true})
	if err != nil {
		t.Fatalf("Log --follow: %v", err)
	}
	if len(plain) != 1 {
		t.Errorf("plain log len = %d, want 1 (rename commit only)", len(plain))
	}
	if len(followed) != 2 {
		t.Errorf("follow log len = %d, want 2 (rename + original)", len(followed))
	}

	scoped, err := n.Log(ctx, git.LogOptions{File: "new.txt", Branch: "main", Follow: true})
	if err != nil {
		t.Fatalf("Log branch-scoped follow: %v", err)
	}
	if len(scoped) != 2 {
		t.Errorf("branch-scoped follow len = %d, want 2", len(scoped))
	}

	if _, err := n.Log(ctx, git.LogOptions{Follow: true}); err == nil {
		t.Errorf("Log(follow, no file) = nil, want error")
	}
}

func TestLogPickaxe(t *testing.T) {
	gitTestPath(t)
	dir := t.TempDir()
	init := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	init("init", "-q", "-b", "main")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "code.txt"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha\n")
	runGit(t, dir, "A", "a@ex.com", "add", ".")
	runGit(t, dir, "A", "a@ex.com", "commit", "-qm", "first")
	write("alpha\nneedle here\n")
	runGit(t, dir, "A", "a@ex.com", "commit", "-qam", "add needle")
	write("alpha\nneedle here\ntrailing\n")
	runGit(t, dir, "A", "a@ex.com", "commit", "-qam", "third")

	n, ctx := openTestRepo(t, dir)

	found, err := n.Log(ctx, git.LogOptions{Pickaxe: "needle"})
	if err != nil {
		t.Fatalf("Log -S: %v", err)
	}
	if len(found) != 1 || found[0].Message != "add needle" {
		t.Errorf("-Sneedle found %d commits, want exactly [add needle]", len(found))
	}

	foundR, err := n.Log(ctx, git.LogOptions{PickaxeR: "needle.*"})
	if err != nil {
		t.Fatalf("Log -G: %v", err)
	}
	if len(foundR) != 1 || foundR[0].Message != "add needle" {
		t.Errorf("-Gneedle found %d commits, want exactly [add needle]", len(foundR))
	}
}

func TestListFiles(t *testing.T) {
	repo := initBlameRepo(t)
	n, ctx := openTestRepo(t, repo)

	files, err := n.ListFiles(ctx)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 || files[0] != "notes.txt" {
		t.Errorf("files = %v, want [notes.txt]", files)
	}
}
