package core

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// scanSkipDirs are never descended into (dependency caches, build output).
var scanSkipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".cache":       true,
	"__pycache__":  true,
	".venv":        true,
	"target":       true,
	"dist":         true,
	"build":        true,
	".Trash":       true,
}

// maxScanResults caps a scan so a huge tree can't flood the UI.
const maxScanResults = 500

// ScanForRepos walks root (maxDepth levels deep) and returns every git
// repository found. A repo is a directory containing .git (file or dir,
// so worktrees and submodules count) or a bare repo layout.
// Results are sorted by path; Tracked flags entries already in the app.
func (e *Engine) ScanForRepos(root string, maxDepth int) ([]models.ScannedRepo, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "scan", Path: root, Err: errors.New("not a directory")}
	}
	if maxDepth < 1 {
		maxDepth = 1
	}
	if maxDepth > 6 {
		maxDepth = 6
	}

	tracked := map[string]bool{}
	if e.Store != nil {
		for _, p := range e.Store.DashboardRepos() {
			tracked[p] = true
		}
	}

	var out []models.ScannedRepo
	rootDepth := len(splitPath(root))
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: skip, don't fail the scan
		}
		if !d.IsDir() {
			return nil
		}
		if len(splitPath(path))-rootDepth > maxDepth {
			return fs.SkipDir
		}
		if path != root && scanSkipDirs[d.Name()] {
			return fs.SkipDir
		}
		if isGitRepoDir(path) {
			out = append(out, models.ScannedRepo{
				Path:    path,
				Name:    filepath.Base(path),
				Tracked: tracked[path],
			})
			if len(out) >= maxScanResults {
				return fs.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// TrackRepos records several repos at once, returning how many were new.
// Unknown/invalid paths are skipped, never fatal.
func (e *Engine) TrackRepos(paths []string) int {
	n := 0
	seen := map[string]bool{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			continue
		}
		e.TrackRepo(abs)
		n++
	}
	return n
}

// isGitRepoDir reports a working tree (.git file or dir) or bare layout.
func isGitRepoDir(path string) bool {
	if st, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return st.IsDir() || !st.IsDir() // dir (clone) or file (worktree/submodule)
	}
	// Bare: HEAD + objects + refs directly inside.
	for _, m := range []string{"HEAD", "objects", "refs"} {
		if _, err := os.Stat(filepath.Join(path, m)); err != nil {
			return false
		}
	}
	return true
}

func splitPath(p string) []string {
	return strings.Split(filepath.Clean(p), string(filepath.Separator))
}
