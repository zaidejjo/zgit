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
}
