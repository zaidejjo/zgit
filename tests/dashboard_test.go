package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/config"
	"github.com/zaidejjo/zgit/pkg/core/db"
)

func testEngine(t *testing.T) *core.Engine {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cfg, err := config.New()
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(filepath.Join(t.TempDir(), "zgit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return &core.Engine{Config: cfg, Store: store}
}

func TestDashboardReposOrdering(t *testing.T) {
	e := testEngine(t)
	e.TrackRepo("/r/a")
	e.TrackRepo("/r/b")
	e.TrackRepo("/r/a") // recency bump
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

	ov := e.RepoOverview(repo, true)
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
	// Second call serves the cached snapshot.
	cached := e.RepoOverview(repo, false)
	if cached.DirtyCount != ov.DirtyCount || cached.Branch != ov.Branch {
		t.Errorf("cached snapshot differs: %+v vs %+v", cached, ov)
	}
	ov = e.RepoOverview(repo, true)
	if ov.IsClean {
		t.Errorf("want dirty after new untracked file")
	}

	// Missing path comes back as an error card, not a failure.
	ov = e.RepoOverview(filepath.Join(t.TempDir(), "nope"), true)
	if ov.Error == "" {
		t.Errorf("want Error set for missing path")
	}
}

func TestRepoOverviewParallel(t *testing.T) {
	e := testEngine(t)
	repos := []string{initBlameRepo(t), initBlameRepo(t), initBlameRepo(t)}

	// Parallel snapshots must never SQLITE_BUSY: the store serializes
	// on a single connection. Regression test for the "database is
	// locked" warnings on the dashboard.
	errs := make([]string, len(repos))
	var wg sync.WaitGroup
	for i, r := range repos {
		wg.Add(1)
		go func(i int, r string) {
			defer wg.Done()
			if ov := e.RepoOverview(r, true); ov.Error != "" {
				errs[i] = ov.Error
			}
		}(i, r)
	}
	wg.Wait()
	for i, msg := range errs {
		if msg != "" {
			t.Errorf("repo %d overview error = %q", i, msg)
		}
	}
}

func TestPruneDeadRepos(t *testing.T) {
	e := testEngine(t)
	alive := t.TempDir()
	dead := filepath.Join(t.TempDir(), "gone")
	e.TrackRepo(alive)
	e.TrackRepo(dead)
	removed, err := e.PruneDeadRepos()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	for _, p := range e.DashboardRepos() {
		if p == dead {
			t.Errorf("dead path still listed")
		}
	}
}

func TestEngineMigratesLegacyRecents(t *testing.T) {
	gitTestPath(t)
	t.Setenv("HOME", t.TempDir())

	// Seed legacy viper state before the engine exists.
	cfg, err := config.New()
	if err != nil {
		t.Fatal(err)
	}
	cfg.AddRecentRepo("/r/legacy-a")
	cfg.AddRecentRepo("/r/legacy-b")
	cfg.Set("favorite_repos", []string{"/r/legacy-b"})
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	e, err := core.New(core.DefaultEngineOptions())
	if err != nil {
		t.Fatal(err)
	}
	got := e.DashboardRepos()
	if len(got) != 2 || got[0] != "/r/legacy-b" || got[1] != "/r/legacy-a" {
		t.Fatalf("migrated = %v, want [/r/legacy-b /r/legacy-a]", got)
	}

	// Second boot must not duplicate or lose anything.
	e2, err := core.New(core.DefaultEngineOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got2 := e2.DashboardRepos(); len(got2) != 2 {
		t.Fatalf("second boot = %v, want 2 entries", got2)
	}
}

func TestScanForRepos(t *testing.T) {
	e := testEngine(t)
	root := t.TempDir()
	mk := func(rel string) {
		if err := os.MkdirAll(filepath.Join(root, rel, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	mk("a")                 // depth 1 repo
	mk("grp/b")             // depth 2 repo
	mk("grp/deep/c")        // depth 3 repo
	mk("node_modules/skip") // skipped dir
	if err := os.MkdirAll(filepath.Join(root, "plain"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := e.ScanForRepos(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, r := range got {
		paths[r.Path] = true
	}
	for _, want := range []string{"a", "grp/b"} {
		if !paths[filepath.Join(root, want)] {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if paths[filepath.Join(root, "grp/deep/c")] {
		t.Errorf("depth-3 repo should be cut at maxDepth 2")
	}
	if paths[filepath.Join(root, "node_modules/skip")] {
		t.Errorf("node_modules should be skipped")
	}

	// Tracked flag + TrackRepos.
	e.TrackRepo(filepath.Join(root, "a"))
	got, _ = e.ScanForRepos(root, 2)
	for _, r := range got {
		if r.Name == "a" && !r.Tracked {
			t.Errorf("a should be flagged tracked")
		}
	}
	if n := e.TrackRepos([]string{filepath.Join(root, "grp/b"), "/nonexistent", ""}); n != 1 {
		t.Errorf("TrackRepos = %d, want 1", n)
	}
}
