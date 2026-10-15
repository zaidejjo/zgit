package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/db"
)

func openTestStore(t *testing.T) *db.Store {
	t.Helper()
	s, err := db.Open(filepath.Join(t.TempDir(), "zgit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStoreRepos(t *testing.T) {
	s := openTestStore(t)

	if s.Migrated() {
		t.Errorf("fresh store should not be marked migrated")
	}
	if err := s.ImportRecents([]string{"/r/old", "/r/new"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ImportFavorites([]string{"/r/fav"}); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkMigrated(); err != nil {
		t.Fatal(err)
	}
	if !s.Migrated() {
		t.Errorf("should be marked migrated")
	}

	got := s.DashboardRepos()
	want := []string{"/r/fav", "/r/new", "/r/old"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// Track bumps recency.
	if err := s.TrackRepo("/r/old"); err != nil {
		t.Fatal(err)
	}
	if got := s.DashboardRepos(); got[1] != "/r/old" {
		t.Errorf("after track: %v", got)
	}

	// Favorites toggle round-trip.
	if !s.IsFavorite("/r/fav") {
		t.Errorf("/r/fav should be favorite")
	}
	if err := s.SetFavorite("/r/new", true); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFavorite("/r/fav", false); err != nil {
		t.Fatal(err)
	}
	if s.IsFavorite("/r/fav") || !s.IsFavorite("/r/new") {
		t.Errorf("favorite flags wrong")
	}

	// Remove forgets.
	if err := s.RemoveRepo("/r/old"); err != nil {
		t.Fatal(err)
	}
	for _, p := range s.DashboardRepos() {
		if p == "/r/old" {
			t.Errorf("/r/old still listed")
		}
	}
}

func TestStorePruneDead(t *testing.T) {
	s := openTestStore(t)
	alive := t.TempDir()
	dead := filepath.Join(t.TempDir(), "gone")
	favDead := filepath.Join(t.TempDir(), "fav-gone")
	for _, p := range []string{alive, dead, favDead} {
		if err := s.TrackRepo(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetFavorite(favDead, true); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dead, 0755); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(dead)

	removed, err := s.PruneDead(func(p string) bool {
		_, statErr := os.Stat(p)
		return statErr == nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1 (favorite kept)", removed)
	}
	if !s.IsFavorite(favDead) {
		t.Errorf("favorite dead path should be kept")
	}
}

func TestStoreProjects(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateProject("  ", ""); err == nil {
		t.Errorf("empty name should fail")
	}
	id, err := s.CreateProject("work", "#89b4fa")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddProjectRepo(id, "/r/a"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddProjectRepo(id, "/r/b"); err != nil {
		t.Fatal(err)
	}
	// Idempotent re-add keeps order.
	if err := s.AddProjectRepo(id, "/r/a"); err != nil {
		t.Fatal(err)
	}
	projects := s.ListProjects()
	if len(projects) != 1 || projects[0].Name != "work" {
		t.Fatalf("projects = %+v", projects)
	}
	if len(projects[0].Repos) != 2 || projects[0].Repos[0] != "/r/a" {
		t.Errorf("members = %v", projects[0].Repos)
	}

	// Reorder wholesale.
	if err := s.SetProjectRepos(id, []string{"/r/b", "/r/a", "/r/b"}); err != nil {
		t.Fatal(err)
	}
	if got := s.ProjectRepos(id); len(got) != 2 || got[0] != "/r/b" {
		t.Errorf("reordered = %v", got)
	}

	if err := s.RenameProject(id, ""); err == nil {
		t.Errorf("empty rename should fail")
	}
	if err := s.RenameProject(id, "play"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveProjectRepo(id, "/r/b"); err != nil {
		t.Fatal(err)
	}
	if got := s.ProjectRepos(id); len(got) != 1 || got[0] != "/r/a" {
		t.Errorf("after remove = %v", got)
	}

	// Unknown project errors.
	for _, err := range []error{
		func() error { _, e := s.CreateProject("", ""); return e }(),
		s.RenameProject("nope", "x"),
		s.DeleteProject("nope"),
		s.AddProjectRepo("nope", "/r/a"),
		s.SetProjectRepos("nope", nil),
	} {
		if err == nil {
			t.Errorf("expected error for unknown project")
		}
	}

	if err := s.DeleteProject(id); err != nil {
		t.Fatal(err)
	}
	if len(s.ListProjects()) != 0 {
		t.Errorf("project should be gone")
	}

	// Removing a repo drops its memberships too.
	id2, _ := s.CreateProject("p2", "")
	_ = s.AddProjectRepo(id2, "/r/gone")
	if err := s.RemoveRepo("/r/gone"); err != nil {
		t.Fatal(err)
	}
	if len(s.ProjectRepos(id2)) != 0 {
		t.Errorf("membership should cascade")
	}
}

func TestStoreProjectColor(t *testing.T) {
	s := openTestStore(t)
	id, err := s.CreateProject("work", "#60a5fa")
	if err != nil {
		t.Fatal(err)
	}
	if got := s.ListProjects(); len(got) != 1 || got[0].Color != "#60a5fa" {
		t.Fatalf("color = %+v", got)
	}
	if err := s.SetProjectColor(id, "#f472b6"); err != nil {
		t.Fatal(err)
	}
	if got := s.ListProjects(); got[0].Color != "#f472b6" {
		t.Errorf("after set: color = %q", got[0].Color)
	}
	if err := s.SetProjectColor("nope", "#fff"); err == nil {
		t.Errorf("expected error for unknown project")
	}
}
