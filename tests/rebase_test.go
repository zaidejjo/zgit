package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// initRebaseRepo creates a repo with three commits and returns dir + oldest-first SHAs.
func initRebaseRepo(t *testing.T) (string, []string) {
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
	var shas []string
	for _, msg := range []string{"one", "two", "three"} {
		// Separate file per commit so reordering never conflicts.
		if err := os.WriteFile(filepath.Join(dir, msg+".txt"), []byte(msg+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		run("add", ".")
		run("commit", "-qm", msg)
		out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal(err)
		}
		shas = append(shas, string(out[:40]))
	}
	return dir, shas
}

func TestRebaseSequenceReorder(t *testing.T) {
	dir, shas := initRebaseRepo(t)
	n, ctx := openTestRepo(t, dir)

	// Swap the last two commits: pick "three" then "two" onto "one".
	res, err := n.RebaseSequence(ctx, models.RebaseSequenceOptions{
		Onto: shas[0],
		Commits: []models.RebaseCommitOp{
			{SHA: shas[2], Action: models.RebasePick},
			{SHA: shas[1], Action: models.RebasePick},
		},
	})
	if err != nil {
		t.Fatalf("RebaseSequence: %v", err)
	}
	if !res.Success {
		t.Fatalf("rebase failed: %s", res.Message)
	}

	// Newest-first log must now read two, three, one.
	out, err := exec.Command("git", "-C", dir, "log", "--format=%s", "-3").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "two\nthree\none\n"
	if string(out) != want {
		t.Errorf("log after reorder =\n%s\nwant\n%s", out, want)
	}
}
