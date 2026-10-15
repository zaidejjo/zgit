package core

import (
	"context"
	"os"
	"path/filepath"
	"time"

	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// DashboardRepos returns favorite repos first, then recent ones (deduped).
func (e *Engine) DashboardRepos() []string {
	seen := make(map[string]bool)
	var out []string
	for _, p := range e.Config.GetFavoriteRepos() {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, p := range e.Config.GetRecentRepos() {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// ToggleFavoriteRepo pins or unpins a repo and persists the change.
// Returns true when the repo is now pinned.
func (e *Engine) ToggleFavoriteRepo(path string) (bool, error) {
	pinned := e.Config.ToggleFavoriteRepo(path)
	if err := e.Config.Save(); err != nil {
		return pinned, err
	}
	return pinned, nil
}

// RepoOverview builds a one-glance status card for a local repo path.
// It never fails: unreadable paths come back with Error set so the UI
// can show dead entries instead of dropping them silently.
func (e *Engine) RepoOverview(repoPath string) *models.RepoOverview {
	ov := &models.RepoOverview{
		Path:     repoPath,
		Name:     filepath.Base(repoPath),
		Favorite: e.Config.IsFavoriteRepo(repoPath),
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

// PruneDeadRepos drops recent entries whose paths no longer exist on disk.
// Favorites are kept (they may live on unmounted drives). Returns the
// number of entries removed.
func (e *Engine) PruneDeadRepos() (int, error) {
	kept := make([]string, 0)
	removed := 0
	for _, p := range e.Config.GetRecentRepos() {
		if _, err := os.Stat(p); err != nil {
			removed++
			continue
		}
		kept = append(kept, p)
	}
	if removed > 0 {
		e.Config.Set("recent_repos", kept)
		if err := e.Config.Save(); err != nil {
			return removed, err
		}
	}
	return removed, nil
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
