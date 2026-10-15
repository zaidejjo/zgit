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
