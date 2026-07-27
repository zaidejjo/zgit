package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"golang.org/x/oauth2"
)

// RESTClient implements GitHubClient using the go-github REST SDK.
type RESTClient struct {
	client *github.Client
	user   string // cached authenticated user login
}

// NewRESTClient creates a RESTClient with the given personal access token.
func NewRESTClient(token string) (*RESTClient, error) {
	if token == "" {
		return nil, fmt.Errorf("github token is required")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	return &RESTClient{client: client}, nil
}

// NewRESTClientWithClient creates a RESTClient with an existing HTTP client (for testing).
func NewRESTClientWithClient(client *github.Client) *RESTClient {
	return &RESTClient{client: client}
}

// --- Auth ---

func (c *RESTClient) GetAuthenticatedUser(ctx context.Context) (*models.User, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	c.user = user.GetLogin()
	return convertUser(user), nil
}

func (c *RESTClient) TestToken(ctx context.Context) error {
	_, _, err := c.client.Users.Get(ctx, "")
	return err
}

// --- Repos ---

func (c *RESTClient) GetRepository(ctx context.Context, owner, name string) (*models.Repo, error) {
	repo, _, err := c.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("get repository %s/%s: %w", owner, name, err)
	}
	return convertRepo(repo), nil
}

func (c *RESTClient) ListRepositories(ctx context.Context) ([]*models.Repo, error) {
	opts := &github.RepositoryListOptions{
		Type:        "owner",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 50},
	}

	var all []*models.Repo
	for {
		repos, resp, err := c.client.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, fmt.Errorf("list repos: %w", err)
		}
		for _, r := range repos {
			all = append(all, convertRepo(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// --- Issues ---

func (c *RESTClient) ListIssues(ctx context.Context, owner, repo string, opts IssuesFilter) ([]*models.Issue, error) {
	opt := &github.IssueListByRepoOptions{
		State:       opts.State,
		Sort:        opts.Sort,
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 50},
	}
	if opts.Label != "" {
		opt.Labels = []string{opts.Label}
	}
	if opts.Author != "" {
		opt.Creator = opts.Author
	}
	if opts.Assignee != "" {
		opt.Assignee = opts.Assignee
	}

	var all []*models.Issue
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list issues: %w", err)
		}
		for _, i := range issues {
			if i.IsPullRequest() {
				continue // skip PRs
			}
			all = append(all, convertIssue(i))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		opt.Page = resp.NextPage
	}
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	return all, nil
}

func (c *RESTClient) GetIssue(ctx context.Context, owner, repo string, number int) (*models.Issue, error) {
	issue, _, err := c.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("get issue #%d: %w", number, err)
	}
	return convertIssue(issue), nil
}

func (c *RESTClient) CreateIssue(ctx context.Context, owner, repo string, req IssueRequest) (*models.Issue, error) {
	ir := &github.IssueRequest{
		Title: &req.Title,
		Body:  &req.Body,
	}
	if len(req.Assignees) > 0 {
		ir.Assignees = &req.Assignees
	}
	if len(req.Labels) > 0 {
		ir.Labels = &req.Labels
	}

	issue, _, err := c.client.Issues.Create(ctx, owner, repo, ir)
	if err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	return convertIssue(issue), nil
}

func (c *RESTClient) UpdateIssue(ctx context.Context, owner, repo string, number int, req IssueRequest) (*models.Issue, error) {
	ir := &github.IssueRequest{
		Title: &req.Title,
		Body:  &req.Body,
	}
	if len(req.Assignees) > 0 {
		ir.Assignees = &req.Assignees
	}
	if len(req.Labels) > 0 {
		ir.Labels = &req.Labels
	}
	if req.Milestone > 0 {
		ir.Milestone = &req.Milestone
	}

	issue, _, err := c.client.Issues.Edit(ctx, owner, repo, number, ir)
	if err != nil {
		return nil, fmt.Errorf("update issue #%d: %w", number, err)
	}
	return convertIssue(issue), nil
}

func (c *RESTClient) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	state := "closed"
	ir := &github.IssueRequest{State: &state}
	_, _, err := c.client.Issues.Edit(ctx, owner, repo, number, ir)
	return err
}

func (c *RESTClient) IssueComments(ctx context.Context, owner, repo string, number int) ([]*models.IssueComment, error) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}
	var all []*models.IssueComment
	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, owner, repo, number, opt)
		if err != nil {
			return nil, fmt.Errorf("list comments for #%d: %w", number, err)
		}
		for _, c := range comments {
			all = append(all, convertComment(c))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// --- Pull Requests (REST) ---

func (c *RESTClient) ListPullRequests(ctx context.Context, owner, repo string, opts PRFilter) ([]*models.PullRequestSummary, error) {
	opt := &github.PullRequestListOptions{
		State:       opts.State,
		Sort:        opts.Sort,
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 50},
	}
	if opts.Head != "" {
		opt.Head = opts.Head
	}
	if opts.Base != "" {
		opt.Base = opts.Base
	}

	var all []*models.PullRequestSummary
	for {
		prs, resp, err := c.client.PullRequests.List(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list PRs: %w", err)
		}
		for _, pr := range prs {
			all = append(all, convertPRSummary(pr))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		opt.Page = resp.NextPage
	}
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	return all, nil
}

func (c *RESTClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("get PR #%d: %w", number, err)
	}

	detail := convertPRDetail(pr)

	// Fetch reviews
	reviews, err := c.listReviews(ctx, owner, repo, number)
	if err == nil {
		detail.Reviews = reviews
	}

	return detail, nil
}

func (c *RESTClient) CreatePullRequest(ctx context.Context, owner, repo string, req PRRequest) (*models.PullRequestSummary, error) {
	pr := &github.NewPullRequest{
		Title: &req.Title,
		Body:  &req.Body,
		Head:  &req.Head,
		Base:  &req.Base,
	}
	if req.Draft {
		draft := true
		pr.Draft = &draft
	}

	created, _, err := c.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	return convertPRSummary(created), nil
}

func (c *RESTClient) MergePullRequest(ctx context.Context, owner, repo string, number int, method string) error {
	opts := &github.PullRequestOptions{MergeMethod: method}
	_, _, err := c.client.PullRequests.Merge(ctx, owner, repo, number, "", opts)
	return err
}

func (c *RESTClient) RequestReview(ctx context.Context, owner, repo string, number int, reviewers []string) error {
	_, _, err := c.client.PullRequests.RequestReviewers(ctx, owner, repo, number, github.ReviewersRequest{Reviewers: reviewers})
	return err
}

func (c *RESTClient) listReviews(ctx context.Context, owner, repo string, number int) ([]models.Review, error) {
	opt := &github.ListOptions{PerPage: 50}
	var all []models.Review
	for {
		reviews, resp, err := c.client.PullRequests.ListReviews(ctx, owner, repo, number, opt)
		if err != nil {
			return nil, err
		}
		for _, r := range reviews {
			all = append(all, convertReview(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// --- Workflows ---

func (c *RESTClient) ListWorkflows(ctx context.Context, owner, repo string) ([]*models.Workflow, error) {
	opt := &github.ListOptions{PerPage: 50}
	var all []*models.Workflow
	for {
		workflows, resp, err := c.client.Actions.ListWorkflows(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list workflows: %w", err)
		}
		for _, w := range workflows.Workflows {
			all = append(all, convertWorkflow(w))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

func (c *RESTClient) ListWorkflowRuns(ctx context.Context, owner, repo string, opts RunsFilter) ([]*models.WorkflowRun, error) {
	opt := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}
	if opts.Branch != "" {
		opt.Branch = opts.Branch
	}
	if opts.Event != "" {
		opt.Event = opts.Event
	}
	if opts.Status != "" {
		opt.Status = opts.Status
	}

	var all []*models.WorkflowRun
	for {
		runs, resp, err := c.client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list workflow runs: %w", err)
		}
		for _, r := range runs.WorkflowRuns {
			all = append(all, convertWorkflowRun(r))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		opt.Page = resp.NextPage
	}
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	return all, nil
}

func (c *RESTClient) GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*models.WorkflowRun, error) {
	run, _, err := c.client.Actions.GetWorkflowRunByID(ctx, owner, repo, runID)
	if err != nil {
		return nil, fmt.Errorf("get workflow run %d: %w", runID, err)
	}
	return convertWorkflowRun(run), nil
}

func (c *RESTClient) ReRunWorkflow(ctx context.Context, owner, repo string, runID int64) error {
	_, err := c.client.Actions.RerunWorkflowByID(ctx, owner, repo, runID)
	return err
}

func (c *RESTClient) CancelWorkflowRun(ctx context.Context, owner, repo string, runID int64) error {
	_, err := c.client.Actions.CancelWorkflowRunByID(ctx, owner, repo, runID)
	return err
}

func (c *RESTClient) GetWorkflowJobLogs(ctx context.Context, owner, repo string, jobID int64) (string, error) {
	url, resp, err := c.client.Actions.GetWorkflowJobLogs(ctx, owner, repo, jobID, 10) // 10MB max
	if err != nil {
		return "", fmt.Errorf("get job logs URL for job %d: %w", jobID, err)
	}
	defer resp.Body.Close()

	// Fetch the log content from the redirect URL
	logReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create log request: %w", err)
	}
	logResp, err := http.DefaultClient.Do(logReq)
	if err != nil {
		return "", fmt.Errorf("fetch logs: %w", err)
	}
	defer logResp.Body.Close()

	data, err := io.ReadAll(logResp.Body)
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}
	return string(data), nil
}

func (c *RESTClient) ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64) ([]*models.Job, error) {
	opt := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}
	var all []*models.Job
	for {
		jobs, resp, err := c.client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opt)
		if err != nil {
			return nil, fmt.Errorf("list jobs for run %d: %w", runID, err)
		}
		for _, j := range jobs.Jobs {
			all = append(all, convertJob(j))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// --- Composite (GraphQL stubs for REST-only mode) ---

func (c *RESTClient) GetDashboard(ctx context.Context) (*models.Dashboard, error) {
	// REST-only dashboard uses multiple API calls
	return nil, fmt.Errorf("GetDashboard requires GraphQL; use GetPullRequestGraphQL or a combined client")
}

func (c *RESTClient) GetPullRequestGraphQL(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error) {
	// Fall back to REST for PR details when GraphQL is unavailable
	return nil, fmt.Errorf("GetPullRequestGraphQL requires GraphQL; use GetPullRequest for REST-only")
}

// --- Converters ---

func convertUser(u *github.User) *models.User {
	if u == nil {
		return nil
	}
	return &models.User{
		Login:       u.GetLogin(),
		Name:        u.GetName(),
		Email:       u.GetEmail(),
		AvatarURL:   u.GetAvatarURL(),
		Bio:         u.GetBio(),
		Company:     u.GetCompany(),
		Location:    u.GetLocation(),
		Followers:   u.GetFollowers(),
		Following:   u.GetFollowing(),
		PublicRepos: u.GetPublicRepos(),
	}
}

func convertRepo(r *github.Repository) *models.Repo {
	if r == nil {
		return nil
	}
	return &models.Repo{
		Owner:         r.GetOwner().GetLogin(),
		Name:          r.GetName(),
		FullName:      r.GetFullName(),
		DefaultBranch: r.GetDefaultBranch(),
		Description:   r.GetDescription(),
		Language:      r.GetLanguage(),
		IsPrivate:     r.GetPrivate(),
		IsFork:        r.GetFork(),
		Stars:         r.GetStargazersCount(),
		Forks:         r.GetForksCount(),
		OpenIssues:    r.GetOpenIssuesCount(),
		CreatedAt:     r.GetCreatedAt().Time,
		UpdatedAt:     r.GetUpdatedAt().Time,
		HTMLURL:       r.GetHTMLURL(),
		SSHURL:        r.GetSSHURL(),
	}
}

func convertIssue(i *github.Issue) *models.Issue {
	if i == nil {
		return nil
	}
	labels := make([]models.Label, 0, len(i.Labels))
	for _, l := range i.Labels {
		labels = append(labels, models.Label{
			Name:  l.GetName(),
			Color: l.GetColor(),
		})
	}
	assignees := make([]string, 0, len(i.Assignees))
	for _, a := range i.Assignees {
		assignees = append(assignees, a.GetLogin())
	}

	state := models.IssueOpen
	if i.GetState() == "closed" {
		state = models.IssueClosed
	}

	return &models.Issue{
		Number:    i.GetNumber(),
		Title:     i.GetTitle(),
		State:     state,
		Author:    i.GetUser().GetLogin(),
		Body:      i.GetBody(),
		CreatedAt: i.GetCreatedAt().Time,
		UpdatedAt: i.GetUpdatedAt().Time,
		Labels:    labels,
		Assignees: assignees,
		Comments:  i.GetComments(),
	}
}

func convertComment(c *github.IssueComment) *models.IssueComment {
	if c == nil {
		return nil
	}
	return &models.IssueComment{
		ID:        c.GetID(),
		Author:    c.GetUser().GetLogin(),
		Body:      c.GetBody(),
		CreatedAt: c.GetCreatedAt().Time,
		UpdatedAt: c.GetUpdatedAt().Time,
	}
}

func convertPRSummary(pr *github.PullRequest) *models.PullRequestSummary {
	if pr == nil {
		return nil
	}
	labels := make([]string, 0, len(pr.Labels))
	for _, l := range pr.Labels {
		labels = append(labels, l.GetName())
	}

	state := models.PRStateOpen
	switch pr.GetState() {
	case "closed":
		if pr.GetMerged() {
			state = models.PRStateMerged
		} else {
			state = models.PRStateClosed
		}
	case "open":
		if pr.GetDraft() {
			state = models.PRStateDraft
		}
	}

	return &models.PullRequestSummary{
		Number:    pr.GetNumber(),
		Title:     pr.GetTitle(),
		State:     state,
		Author:    pr.GetUser().GetLogin(),
		CreatedAt: pr.GetCreatedAt().Time,
		UpdatedAt: pr.GetUpdatedAt().Time,
		IsDraft:   pr.GetDraft(),
		Mergeable: pr.GetMergeableState(),
		HeadRef:   pr.GetHead().GetRef(),
		BaseRef:   pr.GetBase().GetRef(),
		Labels:    labels,
	}
}

func convertPRDetail(pr *github.PullRequest) *models.PullRequestDetail {
	if pr == nil {
		return nil
	}
	summary := convertPRSummary(pr)
	if summary == nil {
		return nil
	}
	return &models.PullRequestDetail{
		PullRequestSummary: *summary,
		Body:               pr.GetBody(),
		Additions:          pr.GetAdditions(),
		Deletions:          pr.GetDeletions(),
		ChangedFiles:       pr.GetChangedFiles(),
		Comments:           pr.GetComments(),
	}
}

func convertReview(r *github.PullRequestReview) models.Review {
	state := models.ReviewPending
	switch r.GetState() {
	case "APPROVED":
		state = models.ReviewApproved
	case "CHANGES_REQUESTED":
		state = models.ReviewChangesRequested
	case "COMMENTED":
		state = models.ReviewCommented
	case "DISMISSED":
		state = models.ReviewDismissed
	}
	return models.Review{
		ID:        r.GetID(),
		Author:    r.GetUser().GetLogin(),
		State:     state,
		Body:      r.GetBody(),
		Submitted: r.GetSubmittedAt().Time,
	}
}

func convertWorkflow(w *github.Workflow) *models.Workflow {
	if w == nil {
		return nil
	}
	return &models.Workflow{
		ID:        w.GetID(),
		Name:      w.GetName(),
		Path:      w.GetPath(),
		State:     w.GetState(),
		CreatedAt: w.GetCreatedAt().Time,
		UpdatedAt: w.GetUpdatedAt().Time,
	}
}

func convertWorkflowRun(r *github.WorkflowRun) *models.WorkflowRun {
	if r == nil {
		return nil
	}
	status := models.RunQueued
	switch r.GetStatus() {
	case "queued":
		status = models.RunQueued
	case "in_progress":
		status = models.RunInProgress
	case "completed":
		status = models.RunCompleted
		switch r.GetConclusion() {
		case "success":
			status = models.RunSuccess
		case "failure":
			status = models.RunFailure
		case "cancelled":
			status = models.RunCancelled
		case "skipped":
			status = models.RunSkipped
		case "stale":
			status = models.RunStale
		}
	}

	return &models.WorkflowRun{
		ID:           r.GetID(),
		WorkflowName: r.GetName(),
		Event:        r.GetEvent(),
		Status:       status,
		Conclusion:   r.GetConclusion(),
		Branch:       r.GetHeadBranch(),
		HeadSHA:      r.GetHeadSHA(),
		RunNumber:    r.GetRunNumber(),
		CreatedAt:    r.GetCreatedAt().Time,
		UpdatedAt:    r.GetUpdatedAt().Time,
		HTMLURL:      r.GetHTMLURL(),
	}
}

func convertJob(j *github.WorkflowJob) *models.Job {
	if j == nil {
		return nil
	}
	job := &models.Job{
		ID:          j.GetID(),
		Name:        j.GetName(),
		Status:      j.GetStatus(),
		Conclusion:  j.GetConclusion(),
		StartedAt:   j.GetStartedAt().Time,
		CompletedAt: j.GetCompletedAt().Time,
		RunnerName:  j.GetRunnerName(),
	}
	if j.Steps != nil {
		for _, s := range j.Steps {
			job.Steps = append(job.Steps, models.Step{
				Name:       s.GetName(),
				Status:     s.GetStatus(),
				Conclusion: s.GetConclusion(),
				Number:     int(s.GetNumber()),
			})
		}
	}
	return job
}

// guessOwnerFromRemote extracts owner/repo from a remote URL.
func guessOwnerFromRemote(url string) (string, string, error) {
	url = strings.TrimSuffix(url, ".git")
	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		if len(parts) == 2 {
			segments := strings.Split(strings.TrimSuffix(parts[1], ".git"), "/")
			if len(segments) >= 2 {
				return segments[0], segments[1], nil
			}
		}
	}
	return "", "", fmt.Errorf("cannot parse owner/repo from %q", url)
}
