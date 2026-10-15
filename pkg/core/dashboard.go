package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// DashboardRepos returns favorite repos first, then recent ones (deduped).
func (e *Engine) DashboardRepos() []string {
	if e.Store == nil {
		return nil
	}
	return e.Store.DashboardRepos()
}

// ToggleFavoriteRepo pins or unpins a repo and persists the change.
// Returns true when the repo is now pinned.
func (e *Engine) ToggleFavoriteRepo(path string) (bool, error) {
	if e.Store == nil {
		return false, fmt.Errorf("store not initialized")
	}
	fav := !e.Store.IsFavorite(path)
	if err := e.Store.SetFavorite(path, fav); err != nil {
		return false, err
	}
	return fav, nil
}

// TrackRepo records a repo open (recency + legacy viper mirror).
func (e *Engine) TrackRepo(path string) {
	if e.Store != nil {
		if err := e.Store.TrackRepo(path); err != nil {
			log.Printf("warning: track repo: %v", err)
		}
	}
	e.Config.AddRecentRepo(path)
	if err := e.Config.Save(); err != nil {
		log.Printf("warning: save config after TrackRepo: %v", err)
	}
}

// RemoveRepo forgets a repo in the app (files on disk are untouched).
func (e *Engine) RemoveRepo(path string) error {
	if e.Store != nil {
		if err := e.Store.RemoveRepo(path); err != nil {
			return err
		}
	}
	// Keep the legacy viper list consistent too.
	kept := make([]string, 0)
	for _, r := range e.Config.GetRecentRepos() {
		if r != path {
			kept = append(kept, r)
		}
	}
	e.Config.Set("recent_repos", kept)
	return e.Config.Save()
}

// RepoOverview builds a one-glance status card for a local repo path.
// It never fails: unreadable paths come back with Error set so the UI
// can show dead entries instead of dropping them silently.
func (e *Engine) RepoOverview(repoPath string) *models.RepoOverview {
	fav := false
	if e.Store != nil {
		fav = e.Store.IsFavorite(repoPath)
	}
	ov := &models.RepoOverview{
		Path:     repoPath,
		Name:     filepath.Base(repoPath),
		Favorite: fav,
	}

	git, err := gitpkg.NewNativeExec("")
	if err != nil {
		ov.Error = err.Error()
		return ov
	}
	if err := git.Open(repoPath); err != nil {
		ov.Error = err.Error()
		return ov
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	status, err := git.Status(ctx)
	if err != nil {
		ov.Error = err.Error()
		return ov
	}
	ov.Branch = status.Branch
	ov.Upstream = status.Upstream
	ov.Ahead = status.Ahead
	ov.Behind = status.Behind
	ov.IsClean = status.IsClean
	ov.DirtyCount = len(status.Files)

	// Latest commit is best effort: empty repos have no HEAD yet.
	if commits, err := git.Log(ctx, gitpkg.LogOptions{Count: 1}); err == nil && len(commits) > 0 {
		ov.LastHash = commits[0].Hash
		ov.LastMessage = commits[0].Message
		ov.LastTime = commits[0].Timestamp.Unix()
	}
	return ov
}

// PruneDeadRepos drops entries whose paths no longer exist on disk.
// Favorites are kept (they may live on unmounted drives). Returns the
// number of entries removed.
func (e *Engine) PruneDeadRepos() (int, error) {
	if e.Store == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	removed, err := e.Store.PruneDead(func(p string) bool {
		_, statErr := os.Stat(p)
		return statErr == nil
	})
	if err != nil {
		return removed, err
	}
	// Keep the legacy viper list consistent too.
	kept := make([]string, 0)
	for _, r := range e.Config.GetRecentRepos() {
		if _, statErr := os.Stat(r); statErr == nil {
			kept = append(kept, r)
		}
	}
	e.Config.Set("recent_repos", kept)
	return removed, e.Config.Save()
}

// FetchRepo fetches the default remote of the repo at repoPath (with prune)
// without switching the currently open repository.
func (e *Engine) FetchRepo(repoPath string) error {
	git, err := gitpkg.NewNativeExec("")
	if err != nil {
		return err
	}
	if err := git.Open(repoPath); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return git.Fetch(ctx, "", true)
}
