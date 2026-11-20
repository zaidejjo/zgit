package pcache

import (
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/cache"
	"github.com/zaidejjo/zgit/pkg/core/db"
)

func TestPcacheSetGet(t *testing.T) {
	store, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c := New("/repo1", store, nil)
	val := []byte("hello-world")

	if err := c.Set(ClassGit, "log/HEAD", val); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get(ClassGit, "log/HEAD")
	if !ok {
		t.Fatal("Get() returned false, want true")
	}
	if string(got) != string(val) {
		t.Errorf("got %q, want %q", got, val)
	}
}

func TestPcacheMiss(t *testing.T) {
	store, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c := New("/repo1", store, nil)
	if _, ok := c.Get(ClassGit, "missing"); ok {
		t.Fatal("Get() returned true for missing key")
	}
}

func TestPcacheScopeIsolation(t *testing.T) {
	store, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	r1 := New("/repo1", store, nil)
	r2 := New("/repo2", store, nil)

	if err := r1.Set(ClassGit, "log/HEAD", []byte("r1-data")); err != nil {
		t.Fatal(err)
	}
	if err := r2.Set(ClassGit, "log/HEAD", []byte("r2-data")); err != nil {
		t.Fatal(err)
	}

	got, _ := r1.Get(ClassGit, "log/HEAD")
	if string(got) != "r1-data" {
		t.Errorf("repo1 got %q, want r1-data", got)
	}
	got, _ = r2.Get(ClassGit, "log/HEAD")
	if string(got) != "r2-data" {
		t.Errorf("repo2 got %q, want r2-data", got)
	}
}

func TestPcacheInvalidate(t *testing.T) {
	store, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c := New("/repo1", store, nil)
	if err := c.Set(ClassGitHub, "prs", []byte("pr-data")); err != nil {
		t.Fatal(err)
	}
	c.Invalidate(ClassGitHub, "prs")
	if _, ok := c.Get(ClassGitHub, "prs"); ok {
		t.Fatal("Get() returned true after Invalidate")
	}
}

func TestPcacheNoStore(t *testing.T) {
	mem, _ := cache.New(100, 0)
	c := New("/repo1", nil, mem)
	val := []byte("mem-only")
	if err := c.Set(ClassGit, "x", val); err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get(ClassGit, "x")
	if !ok || string(got) != string(val) {
		t.Fatal("memory-only pcache failed")
	}
}
