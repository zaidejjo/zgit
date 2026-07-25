package github

import (
	"context"
	"fmt"
	"sync"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// CombinedClient implements GitHubClient by delegating to REST and GraphQL backends.
// REST handles CRUD operations; GraphQL handles rich composite queries.
type CombinedClient struct {
	rest       *RESTClient
	graphql    *GraphQLClient
	mu         sync.RWMutex
	cachedUser string
}

// NewCombinedClient creates a client using both REST and GraphQL.
func NewCombinedClient(token string) (*CombinedClient, error) {
	rest, err := NewRESTClient(token)
	if err != nil {
		return nil, err
	}
	graphql := NewGraphQLClient(token)
	return &CombinedClient{rest: rest, graphql: graphql}, nil
}

// NewCombinedClientWithClients creates a CombinedClient with pre-configured clients (for testing).
func NewCombinedClientWithClients(rest *RESTClient, graphql *GraphQLClient) *CombinedClient {
	return &CombinedClient{rest: rest, graphql: graphql}
}

// --- Auth ---

func (c *CombinedClient) GetAuthenticatedUser(ctx context.Context) (*models.User, error) {
	user, err := c.rest.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.cachedUser = user.Login
	c.mu.Unlock()
	return user, nil
}

func (c *CombinedClient) TestToken(ctx context.Context) error {
	return c.rest.TestToken(ctx)
}

// --- Repos ---

func (c *CombinedClient) GetRepository(ctx context.Context, owner, name string) (*models.Repo, error) {
	return c.rest.GetRepository(ctx, owner, name)
}

func (c *CombinedClient) ListRepositories(ctx context.Context) ([]*models.Repo, error) {
	return c.rest.ListRepositories(ctx)
}

// --- Issues ---

func (c *CombinedClient) ListIssues(ctx context.Context, owner, repo string, opts IssuesFilter) ([]*models.Issue, error) {
	return c.rest.ListIssues(ctx, owner, repo, opts)
}

func (c *CombinedClient) GetIssue(ctx context.Context, owner, repo string, number int) (*models.Issue, error) {
	return c.rest.GetIssue(ctx, owner, repo, number)
}

func (c *CombinedClient) CreateIssue(ctx context.Context, owner, repo string, req IssueRequest) (*models.Issue, error) {
	return c.rest.CreateIssue(ctx, owner, repo, req)
}

func (c *CombinedClient) UpdateIssue(ctx context.Context, owner, repo string, number int, req IssueRequest) (*models.Issue, error) {
	return c.rest.UpdateIssue(ctx, owner, repo, number, req)
}

func (c *CombinedClient) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	return c.rest.CloseIssue(ctx, owner, repo, number)
}

func (c *CombinedClient) IssueComments(ctx context.Context, owner, repo string, number int) ([]*models.IssueComment, error) {
	return c.rest.IssueComments(ctx, owner, repo, number)
}

// --- Pull Requests ---

func (c *CombinedClient) ListPullRequests(ctx context.Context, owner, repo string, opts PRFilter) ([]*models.PullRequestSummary, error) {
	return c.rest.ListPullRequests(ctx, owner, repo, opts)
}

func (c *CombinedClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error) {
	// Try GraphQL first for richer data
	detail, err := c.graphql.GetPullRequestGraphQL(ctx, owner, repo, number)
	if err == nil {
		return detail, nil
	}
	// Fall back to REST
	return c.rest.GetPullRequest(ctx, owner, repo, number)
}

func (c *CombinedClient) CreatePullRequest(ctx context.Context, owner, repo string, req PRRequest) (*models.PullRequestSummary, error) {
	return c.rest.CreatePullRequest(ctx, owner, repo, req)
}

func (c *CombinedClient) MergePullRequest(ctx context.Context, owner, repo string, number int, method string) error {
	return c.rest.MergePullRequest(ctx, owner, repo, number, method)
}

func (c *CombinedClient) RequestReview(ctx context.Context, owner, repo string, number int, reviewers []string) error {
	return c.rest.RequestReview(ctx, owner, repo, number, reviewers)
}

// --- Workflows ---

func (c *CombinedClient) ListWorkflows(ctx context.Context, owner, repo string) ([]*models.Workflow, error) {
	return c.rest.ListWorkflows(ctx, owner, repo)
}

func (c *CombinedClient) ListWorkflowRuns(ctx context.Context, owner, repo string, opts RunsFilter) ([]*models.WorkflowRun, error) {
	return c.rest.ListWorkflowRuns(ctx, owner, repo, opts)
}

func (c *CombinedClient) GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*models.WorkflowRun, error) {
	return c.rest.GetWorkflowRun(ctx, owner, repo, runID)
}

func (c *CombinedClient) ReRunWorkflow(ctx context.Context, owner, repo string, runID int64) error {
	return c.rest.ReRunWorkflow(ctx, owner, repo, runID)
}

func (c *CombinedClient) CancelWorkflowRun(ctx context.Context, owner, repo string, runID int64) error {
	return c.rest.CancelWorkflowRun(ctx, owner, repo, runID)
}

func (c *CombinedClient) ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64) ([]*models.Job, error) {
	return c.rest.ListWorkflowJobs(ctx, owner, repo, runID)
}

// --- Composite (GraphQL) ---

func (c *CombinedClient) GetDashboard(ctx context.Context) (*models.Dashboard, error) {
	return c.graphql.GetDashboard(ctx)
}

func (c *CombinedClient) GetPullRequestGraphQL(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error) {
	return c.graphql.GetPullRequestGraphQL(ctx, owner, repo, number)
}

// CachedUser returns the cached authenticated user login (set after GetAuthenticatedUser).
func (c *CombinedClient) CachedUser() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cachedUser
}

// Ensure CombinedClient implements GitHubClient.
var _ GitHubClient = (*CombinedClient)(nil)

// guessCurrentRepo attempts to extract owner/repo from git remotes.
func guessCurrentRepo(gitAdapter interface {
	RemoteList(ctx context.Context) ([]*models.Remote, error)
}) (string, string, error) {
	ctx := context.Background()
	remotes, err := gitAdapter.RemoteList(ctx)
	if err != nil {
		return "", "", fmt.Errorf("list remotes: %w", err)
	}
	for _, r := range remotes {
		if r.Name == "origin" || r.Name == "upstream" {
			owner, repo, err := guessOwnerFromRemote(r.URL)
			if err == nil {
				return owner, repo, nil
			}
		}
	}
	if len(remotes) > 0 {
		return guessOwnerFromRemote(remotes[0].URL)
	}
	return "", "", fmt.Errorf("no git remotes found")
}
