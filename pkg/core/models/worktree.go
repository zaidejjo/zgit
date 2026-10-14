package models

// Worktree represents a single linked working tree from `git worktree list --porcelain`.
type Worktree struct {
	// Path is the filesystem path of the worktree root.
	Path string `json:"path"`
	// Head is the checked-out commit SHA (may be empty for bare entries).
	Head string `json:"head,omitempty"`
	// Branch is the checked-out branch short name (empty when detached/bare).
	Branch string `json:"branch,omitempty"`
	// IsMain is true for the main worktree (first entry from git).
	IsMain bool `json:"is_main"`
	// IsBare is true when the worktree is a bare repository entry.
	IsBare bool `json:"is_bare"`
	// IsDetached is true when HEAD is detached.
	IsDetached bool `json:"is_detached"`
	// Locked is true when the worktree is locked (`git worktree lock`).
	Locked bool `json:"locked"`
	// LockReason is the optional reason passed to `git worktree lock`.
	LockReason string `json:"lock_reason,omitempty"`
	// Prunable is true when git reports the worktree as prunable
	// (e.g. its directory was deleted on disk).
	Prunable bool `json:"prunable"`
}

// WorktreeAddOptions controls `git worktree add` behavior.
type WorktreeAddOptions struct {
	// CreateBranch creates a new branch for the worktree (-b).
	// Ignored when Detach is true.
	CreateBranch bool `json:"create_branch"`
	// Force allows checking out an already-checked-out branch (--force).
	Force bool `json:"force"`
	// Detach checks out HEAD detached (--detach).
	Detach bool `json:"detach"`
}
