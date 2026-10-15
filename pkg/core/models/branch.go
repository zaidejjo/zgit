package models

// BranchType distinguishes local vs remote branches.
type BranchType int

const (
	LocalBranch BranchType = iota
	RemoteBranch
)

// String returns a human-readable label for BranchType.
func (t BranchType) String() string {
	switch t {
	case LocalBranch:
		return "local"
	case RemoteBranch:
		return "remote"
	default:
		return "unknown"
	}
}

// Branch represents a Git branch with its upstream tracking info.
type Branch struct {
	Name       string     `json:"name"`
	FullRef    string     `json:"full_ref"` // e.g. "refs/heads/main"
	Type       BranchType `json:"type"`
	IsHead     bool       `json:"is_head"`               // current HEAD branch
	Upstream   string     `json:"upstream,omitempty"`    // e.g. "origin/main"
	Ahead      int        `json:"ahead,omitempty"`       // commits ahead of upstream
	Behind     int        `json:"behind,omitempty"`      // commits behind upstream
	LatestHash string     `json:"latest_hash,omitempty"` // tip commit hash
	LatestMsg  string     `json:"latest_msg,omitempty"`  // tip commit message
}

// Remote represents a Git remote configuration.
type Remote struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	PushURL string `json:"push_url,omitempty"`
	Type    string `json:"type"` // "fetch" or "push"
}

// CleanupCandidate is a branch safe to delete: fully merged into the base,
// not checked out, and not protected. Protected lists names withheld for
// safety (main, master, dev, the base itself, the checked-out branch).
type CleanupCandidate struct {
	RepoPath    string `json:"repo_path"`
	RepoName    string `json:"repo_name"`
	Branch      string `json:"branch"`
	LastMessage string `json:"last_message,omitempty"`
	LastTime    int64  `json:"last_time,omitempty"`
}

// Protected lists branches withheld with a reason instead of offered.
type Protected struct {
	RepoPath string `json:"repo_path"`
	RepoName string `json:"repo_name"`
	Branch   string `json:"branch"`
	Reason   string `json:"reason"`
}
