// Package git provides an interface for local Git operations
// backed by native os/exec calls to the system git binary.
package git

import (
	"context"
	"fmt"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// LogOptions controls git log output.
type LogOptions struct {
	Count    int    // max commits (0 = all)
	Branch   string // ref to log (empty = HEAD)
	File     string // log for a specific file
	Since    string // ISO date or "1 week ago"
	Until    string // ISO date
	Author   string // filter by author
	All      bool   // include all refs
	Merges   bool   // only merge commits
	NoMerges bool   // exclude merge commits
	Format   string // overrides default format (used internally)
}

// DiffOptions controls git diff output.
type DiffOptions struct {
	A        string // left ref (empty = HEAD)
	B        string // right ref (empty = working tree)
	Cached   bool   // diff staged changes (--cached)
	Context  int    // context lines (default 3)
	Unified  bool   // include full unified diff text
	Pathspec string // limit to files matching pathspec
}

// PullOptions controls git pull behavior.
type PullOptions struct {
	Remote string
	Branch string
	Rebase bool
	FFOnly bool
}

// StashOptions controls git stash operations.
type StashOptions struct {
	KeepIndex        bool
	IncludeUntracked bool
	Staged           bool // --staged
}

// PushOptions controls git push behavior.
type PushOptions struct {
	Remote         string
	Branch         string
	Force          bool
	ForceWithLease bool
	SetUpstream    bool
}

// AddOptions controls git add behavior.
type AddOptions struct {
	IntentToAdd bool // -N
	Force       bool // -f
	All         bool // -A
	Update      bool // -u
}

// CommitOptions controls git commit behavior.
type CommitOptions struct {
	AllowEmpty bool
	Amend      bool
	Signoff    bool
	Message    string // subject line
	Body       string // extended description (passed as separate -m)
}

// GitError wraps errors from native git execution.
type GitError struct {
	Stderr   string
	Args     []string
	ExitCode int
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %v exited with code %d: %s", e.Args, e.ExitCode, e.Stderr)
}

// GitAdapter defines the interface for all local Git operations.
// Implementations must be safe for concurrent use.
type GitAdapter interface {
	// Repository info
	Open(path string) error
	Close() error
	Path() string

	// Working tree
	Status(ctx context.Context) (*models.Status, error)
	Diff(ctx context.Context, opts DiffOptions) (*models.Diff, error)
	DiffFiles(ctx context.Context, opts DiffOptions) ([]models.FileChange, error)
	Show(ctx context.Context, ref string) (*models.Diff, error)

	// Commit history
	Log(ctx context.Context, opts LogOptions) ([]*models.Commit, error)
	RevList(ctx context.Context, ref string, count int) ([]string, error)

	// Branch management
	Branches(ctx context.Context) ([]*models.Branch, error)
	CurrentBranch(ctx context.Context) (string, error)
	BranchCreate(ctx context.Context, name string) error
	BranchDelete(ctx context.Context, name string, force bool) error
	BranchRename(ctx context.Context, oldName, newName string) error

	// Checkout
	Checkout(ctx context.Context, ref string) error
	CreateBranchAndCheckout(ctx context.Context, name string) error

	// Staging
	Add(ctx context.Context, opts AddOptions, files ...string) error
	Reset(ctx context.Context, files ...string) error
	Restore(ctx context.Context, files ...string) error
	RestoreStaged(ctx context.Context, files ...string) error
	ApplyPatch(ctx context.Context, patch string, cached bool) error

	// Committing
	Commit(ctx context.Context, opts CommitOptions) error

	// Remotes
	RemoteList(ctx context.Context) ([]*models.Remote, error)
	RemoteAdd(ctx context.Context, name, url string) error
	RemoteRemove(ctx context.Context, name string) error

	// Sync
	Fetch(ctx context.Context, remote string, prune bool) error
	Pull(ctx context.Context, opts PullOptions) error
	Push(ctx context.Context, opts PushOptions) error

	// Stash
	StashList(ctx context.Context) ([]*models.Stash, error)
	StashPush(ctx context.Context, opts StashOptions, message string) error
	StashPop(ctx context.Context, index int) error
	StashApply(ctx context.Context, index int) error
	StashDrop(ctx context.Context, index int) error

	// Merge
	Merge(ctx context.Context, branch string) (string, error)

	// Misc
	MergeBase(ctx context.Context, a, b string) (string, error)
	RefExists(ctx context.Context, ref string) (bool, error)
	RevParse(ctx context.Context, ref string) (string, error)
	IsWorkTreeClean(ctx context.Context) (bool, error)
}
