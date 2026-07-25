// Package github provides a unified client for the GitHub API,
// combining REST (go-github) and GraphQL (githubv4) backends.
package github

import (
	"context"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// IssuesFilter controls issue list queries.
type IssuesFilter struct {
	State    string // "open", "closed", "all"
	Label    string
	Author   string
	Assignee string
	Sort     string // "created", "updated", "comments"
	Limit    int
}

// PRFilter controls pull request list queries.
type PRFilter struct {
	State  string // "open", "closed", "all"
	Head   string // branch name
	Base   string // branch name
	Sort   string // "created", "updated", "popularity"
	Limit  int
	Author string
	Label  string
}

// RunsFilter controls workflow run list queries.
type RunsFilter struct {
	Branch string
	Event  string // "push", "pull_request", etc.
	Status string // "completed", "in_progress", "queued"
	Limit  int
}

// IssueRequest holds fields for creating/updating an issue.
type IssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone int      `json:"milestone,omitempty"`
}

// PRRequest holds fields for creating a pull request.
type PRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft"`
}

// ReviewRequest holds fields for submitting a PR review.
type ReviewRequest struct {
	Body  string `json:"body"`
	Event string `json:"event"` // "APPROVE", "REQUEST_CHANGES", "COMMENT"
}

// GitHubClient defines the interface for GitHub API operations.
// All methods accept a context for cancellation and timeouts.
type GitHubClient interface {
	// Auth
	GetAuthenticatedUser(ctx context.Context) (*models.User, error)
	TestToken(ctx context.Context) error

	// Repos
	GetRepository(ctx context.Context, owner, name string) (*models.Repo, error)
	ListRepositories(ctx context.Context) ([]*models.Repo, error)

	// Issues (REST)
	ListIssues(ctx context.Context, owner, repo string, opts IssuesFilter) ([]*models.Issue, error)
	GetIssue(ctx context.Context, owner, repo string, number int) (*models.Issue, error)
	CreateIssue(ctx context.Context, owner, repo string, req IssueRequest) (*models.Issue, error)
	UpdateIssue(ctx context.Context, owner, repo string, number int, req IssueRequest) (*models.Issue, error)
	CloseIssue(ctx context.Context, owner, repo string, number int) error
	IssueComments(ctx context.Context, owner, repo string, number int) ([]*models.IssueComment, error)

	// Pull Requests (REST + GraphQL)
	ListPullRequests(ctx context.Context, owner, repo string, opts PRFilter) ([]*models.PullRequestSummary, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error)
	CreatePullRequest(ctx context.Context, owner, repo string, req PRRequest) (*models.PullRequestSummary, error)
	MergePullRequest(ctx context.Context, owner, repo string, number int, method string) error
	RequestReview(ctx context.Context, owner, repo string, number int, reviewers []string) error

	// Workflows (REST)
	ListWorkflows(ctx context.Context, owner, repo string) ([]*models.Workflow, error)
	ListWorkflowRuns(ctx context.Context, owner, repo string, opts RunsFilter) ([]*models.WorkflowRun, error)
	GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*models.WorkflowRun, error)
	ReRunWorkflow(ctx context.Context, owner, repo string, runID int64) error
	CancelWorkflowRun(ctx context.Context, owner, repo string, runID int64) error
	ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64) ([]*models.Job, error)

	// Composite queries (GraphQL)
	GetDashboard(ctx context.Context) (*models.Dashboard, error)
	GetPullRequestGraphQL(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error)
}
