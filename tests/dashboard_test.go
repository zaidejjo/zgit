package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/config"
)

func testEngine(t *testing.T) *core.Engine {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cfg, err := config.New()
	if err != nil {
		t.Fatal(err)
	}
	return &core.Engine{Config: cfg}
}

func TestDashboardReposOrdering(t *testing.T) {
	e := testEngine(t)
	e.Config.AddRecentRepo("/r/a")
	e.Config.AddRecentRepo("/r/b")
	e.Config.AddRecentRepo("/r/a") // dedupe bump
	if _, err := e.ToggleFavoriteRepo("/r/b"); err != nil {
		t.Fatal(err)
	}
	got := e.DashboardRepos()
	want := []string{"/r/b", "/r/a"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	// Toggle off removes the pin.
	if pinned, _ := e.ToggleFavoriteRepo("/r/b"); pinned {
		t.Errorf("second toggle should unpin")
	}
}

func TestRepoOverview(t *testing.T) {
	e := testEngine(t)
	repo := initBlameRepo(t)

	ov := e.RepoOverview(repo)
	if ov.Error != "" {
		t.Fatalf("Error = %q", ov.Error)
	}
	if ov.Name == "" {
		t.Errorf("Name empty")
	}
	if ov.Branch != "main" {
		t.Errorf("Branch = %q, want main", ov.Branch)
	}
	if !ov.IsClean || ov.DirtyCount != 0 {
		t.Errorf("want clean, got dirty=%d", ov.DirtyCount)
	}
	if ov.LastMessage != "edit first line" {
		t.Errorf("LastMessage = %q", ov.LastMessage)
	}

	// Dirty file shows up.
	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// Untracked files are not in ls-files based flows, but status sees them.
	ov = e.RepoOverview(repo)
	if ov.IsClean {
		t.Errorf("want dirty after new untracked file")
	}

	// Missing path comes back as an error card, not a failure.
	ov = e.RepoOverview(filepath.Join(t.TempDir(), "nope"))
	if ov.Error == "" {
		t.Errorf("want Error set for missing path")
	}
}

func TestPruneDeadRepos(t *testing.T) {
	e := testEngine(t)
	alive := t.TempDir()
	dead := filepath.Join(t.TempDir(), "gone")
	e.Config.AddRecentRepo(alive)
	e.Config.AddRecentRepo(dead)
	if err := e.Config.Save(); err != nil {
		t.Fatal(err)
	}
	removed, err := e.PruneDeadRepos()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	for _, p := range e.Config.GetRecentRepos() {
		if p == dead {
			t.Errorf("dead path still listed")
		}
	}
}
