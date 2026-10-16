package models

import "time"

// Repo represents a Git repository (local + GitHub metadata).
type Repo struct {
	// Local info
	Path   string `json:"path"`
	IsBare bool   `json:"is_bare"`

	// GitHub metadata (if connected)
	Owner         string    `json:"owner,omitempty"`
	Name          string    `json:"name,omitempty"`
	FullName      string    `json:"full_name,omitempty"` // owner/name
	DefaultBranch string    `json:"default_branch,omitempty"`
	Description   string    `json:"description,omitempty"`
	Language      string    `json:"language,omitempty"`
	IsPrivate     bool      `json:"is_private"`
	IsFork        bool      `json:"is_fork"`
	Stars         int       `json:"stars"`
	Forks         int       `json:"forks"`
	OpenIssues    int       `json:"open_issues"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	HTMLURL       string    `json:"html_url,omitempty"`
	SSHURL        string    `json:"ssh_url,omitempty"`
}

// User represents a GitHub user/account.
type User struct {
	Login       string `json:"login"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Bio         string `json:"bio,omitempty"`
	Company     string `json:"company,omitempty"`
	Location    string `json:"location,omitempty"`
	Plan        string `json:"plan,omitempty"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	PublicRepos int    `json:"public_repos"`
}

// Dashboard is a multi-section overview for the home screen.
type Dashboard struct {
	MyPRs      []PullRequestSummary `json:"my_prs"`
	ReviewPRs  []PullRequestSummary `json:"review_prs"`
	MyIssues   []Issue              `json:"my_issues"`
	RecentRuns []WorkflowRun        `json:"recent_runs"`
	Repos      []Repo               `json:"repos"`
}

// RepoOverview is a one-glance status card for a local repository.
// Error is set (and the rest left zero) when the path is not a readable repo.
type RepoOverview struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Branch      string `json:"branch,omitempty"`
	Upstream    string `json:"upstream,omitempty"`
	Ahead       int    `json:"ahead"`
	Behind      int    `json:"behind"`
	IsClean     bool   `json:"is_clean"`
	DirtyCount  int    `json:"dirty_count"`
	LastHash    string `json:"last_hash,omitempty"`
	LastMessage string `json:"last_message,omitempty"`
	LastTime    int64  `json:"last_time,omitempty"`
	CommitCount int    `json:"commit_count,omitempty"`
	Favorite    bool   `json:"favorite"`
	Error       string `json:"error,omitempty"`
}

// ScannedRepo is one repository found by a folder scan.
type ScannedRepo struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Tracked bool   `json:"tracked"`
}

// Notification represents a GitHub notification.
type Notification struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason"` // review_requested, mention, assign, etc.
	Unread    bool      `json:"unread"`
	Title     string    `json:"title"`
	Type      string    `json:"type"` // Issue, PullRequest, Commit
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
	RepoName  string    `json:"repo_name"`
	RepoURL   string    `json:"repo_url,omitempty"`
	HTMLURL   string    `json:"html_url,omitempty"`
}

// Project groups repositories into a named workspace.
type Project struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Color string   `json:"color,omitempty"`
	Repos []string `json:"repos"`
}
