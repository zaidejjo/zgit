package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/db"
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
	e.Store.InvalidateOverview(path)
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
// RepoOverview builds a one-glance status card for a local repo path.
// Snapshots younger than the cache TTL are served instantly; force
// recomputes (used by explicit Refresh).
func (e *Engine) RepoOverview(repoPath string, force bool) *models.RepoOverview {
	if !force && e.Store != nil {
		if cached, ok := e.Store.GetCachedOverview(repoPath, db.CachedOverviewTTLSeconds); ok {
			// Favorite is cheap and live — overlay so toggles never look stale.
			cached.Favorite = e.Store.IsFavorite(repoPath)
			return cached
		}
	}
	ov := e.computeOverview(repoPath)
	if e.Store != nil && ov.Error == "" {
		if err := e.Store.SetCachedOverview(ov); err != nil {
			log.Printf("warning: cache overview: %v", err)
		}
	}
	return ov
}

func (e *Engine) computeOverview(repoPath string) *models.RepoOverview {
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

	if n, err := git.CommitCount(ctx, "HEAD"); err == nil {
		ov.CommitCount = n
	}
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

// --- Projects ---

// GetProjects returns all projects with members.
func (e *Engine) GetProjects() []models.Project {
	if e.Store == nil {
		return []models.Project{}
	}
	return e.Store.ListProjects()
}

// CreateProject creates a project and returns its id.
func (e *Engine) CreateProject(name, color string) (string, error) {
	if e.Store == nil {
		return "", fmt.Errorf("store not initialized")
	}
	return e.Store.CreateProject(name, color)
}

// RenameProject renames a project.
func (e *Engine) RenameProject(id, name string) error {
	if e.Store == nil {
		return fmt.Errorf("store not initialized")
	}
	return e.Store.RenameProject(id, name)
}

// DeleteProject removes a project (member repos stay in the app).
func (e *Engine) DeleteProject(id string) error {
	if e.Store == nil {
		return fmt.Errorf("store not initialized")
	}
	return e.Store.DeleteProject(id)
}

// AddProjectRepo adds a repo to a project.
func (e *Engine) AddProjectRepo(id, path string) error {
	if e.Store == nil {
		return fmt.Errorf("store not initialized")
	}
	return e.Store.AddProjectRepo(id, path)
}

// RemoveProjectRepo drops a repo from a project.
func (e *Engine) RemoveProjectRepo(id, path string) error {
	if e.Store == nil {
		return fmt.Errorf("store not initialized")
	}
	return e.Store.RemoveProjectRepo(id, path)
}

// SetProjectRepos replaces a project's membership order.
func (e *Engine) SetProjectRepos(id string, paths []string) error {
	if e.Store == nil {
		return fmt.Errorf("store not initialized")
	}
	return e.Store.SetProjectRepos(id, paths)
}

// FetchProject fetches every member repo. Returns repos fetched;
// the first failure aborts.
func (e *Engine) FetchProject(id string) (int, error) {
	if e.Store == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	count := 0
	for _, p := range e.Store.ProjectRepos(id) {
		if err := e.FetchRepo(p); err != nil {
			return count, fmt.Errorf("fetch %s: %w", p, err)
		}
		count++
	}
	return count, nil
}

// --- Branch cleanup ---

// defaultProtected are never offered for deletion.
var defaultProtected = map[string]bool{"main": true, "master": true, "dev": true}

// CleanupScan returns deletion candidates and withheld branches for one repo.
// base empty defaults to the repo's current branch.
func (e *Engine) CleanupScan(repoPath, base string) (candidates []models.CleanupCandidate, withheld []models.Protected, err error) {
	git, err := gitpkg.NewNativeExec("")
	if err != nil {
		return nil, nil, err
	}
	if err := git.Open(repoPath); err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if strings.TrimSpace(base) == "" {
		base, err = git.CurrentBranch(ctx)
		if err != nil || base == "" || base == "HEAD" {
			return nil, nil, fmt.Errorf("cannot determine base branch")
		}
	}
	current, _ := git.CurrentBranch(ctx)

	merged, err := git.BranchMerged(ctx, base)
	if err != nil {
		return nil, nil, err
	}
	name := filepath.Base(repoPath)
	for _, b := range merged {
		switch {
		case b == base:
			withheld = append(withheld, models.Protected{RepoPath: repoPath, RepoName: name, Branch: b, Reason: "merge base"})
		case b == current:
			withheld = append(withheld, models.Protected{RepoPath: repoPath, RepoName: name, Branch: b, Reason: "checked out"})
		case defaultProtected[b]:
			withheld = append(withheld, models.Protected{RepoPath: repoPath, RepoName: name, Branch: b, Reason: "protected"})
		default:
			c := models.CleanupCandidate{RepoPath: repoPath, RepoName: name, Branch: b}
			if commits, err := git.Log(ctx, gitpkg.LogOptions{Branch: b, Count: 1}); err == nil && len(commits) > 0 {
				c.LastMessage = commits[0].Message
				c.LastTime = commits[0].Timestamp.Unix()
			}
			candidates = append(candidates, c)
		}
	}
	if candidates == nil {
		candidates = []models.CleanupCandidate{}
	}
	if withheld == nil {
		withheld = []models.Protected{}
	}
	return candidates, withheld, nil
}

// DeleteRepoBranch safe-deletes (-d, merged-only) a branch in any repo.
func (e *Engine) DeleteRepoBranch(repoPath, branch string) error {
	git, err := gitpkg.NewNativeExec("")
	if err != nil {
		return err
	}
	if err := git.Open(repoPath); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return git.BranchDelete(ctx, branch, false)
}

// ProjectCleanupScan scans every member repo. Base applies where set;
// empty base resolves per repo.
func (e *Engine) ProjectCleanupScan(id, base string) ([]models.CleanupCandidate, []models.Protected, error) {
	if e.Store == nil {
		return nil, nil, fmt.Errorf("store not initialized")
	}
	var candidates []models.CleanupCandidate
	var withheld []models.Protected
	for _, p := range e.Store.ProjectRepos(id) {
		c, w, err := e.CleanupScan(p, base)
		if err != nil {
			withheld = append(withheld, models.Protected{RepoPath: p, RepoName: filepath.Base(p), Branch: "", Reason: "scan failed: " + err.Error()})
			continue
		}
		candidates = append(candidates, c...)
		withheld = append(withheld, w...)
	}
	if candidates == nil {
		candidates = []models.CleanupCandidate{}
	}
	if withheld == nil {
		withheld = []models.Protected{}
	}
	return candidates, withheld, nil
}
