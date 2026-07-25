package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"golang.org/x/oauth2"
)

// GraphQLClient implements composite GitHub queries using the v4 GraphQL API.
type GraphQLClient struct {
	client *githubv4.Client
}

// NewGraphQLClient creates a GraphQL client with the given token.
func NewGraphQLClient(token string) *GraphQLClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &GraphQLClient{client: githubv4.NewClient(tc)}
}

// NewGraphQLClientWithClient creates a GraphQLClient with an existing HTTP client (for testing).
func NewGraphQLClientWithClient(client *githubv4.Client) *GraphQLClient {
	return &GraphQLClient{client: client}
}

// --- GraphQL query types ---

type gqlPR struct {
	Number  int
	Title   string
	State   string
	IsDraft bool
	Author  struct {
		Login string
	}
	CreatedAt time.Time
	UpdatedAt time.Time
	Mergeable string
	HeadRef   struct{ Name string }
	BaseRef   struct{ Name string }
	Labels    struct {
		Nodes []struct {
			Name string
		}
	} `graphql:"labels(first: 10)"`
	Reviews struct {
		Nodes []struct {
			State     string
			Author    struct{ Login string }
			Body      string
			CreatedAt time.Time
		}
	} `graphql:"reviews(first: 20)"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				OID                 string
				MessageHeadline     string
				AuthoredByCommitter bool
				Author              struct {
					Name  string
					Email string
				}
				CommittedDate time.Time
			}
		}
	} `graphql:"commits(first: 50)"`
	StatusCheckRollup *struct {
		Contexts []struct {
			Typename   string `graphql:"__typename"`
			Name       string
			State      string // SUCCESS, FAILURE, PENDING, etc.
			Conclusion string
			DetailsURL string
		}
	} `graphql:"statusCheckRollup"`
	Files struct {
		Nodes []struct {
			Path       string
			Additions  int
			Deletions  int
			ChangeType string
		}
	} `graphql:"files(first: 100)"`
	Body         string
	Additions    int
	Deletions    int
	ChangedFiles int
	Comments     struct {
		TotalCount int `graphql:"totalCount"`
	}
}

type gqlDashboard struct {
	Viewer struct {
		Login        string
		Name         string
		AvatarURL    string
		PullRequests struct {
			Nodes []gqlPR
		} `graphql:"pullRequests(first: 20, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC})"`
		Issues struct {
			Nodes []gqlIssue
		} `graphql:"issues(first: 20, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC})"`
		Repositories struct {
			Nodes []gqlRepo
		} `graphql:"repositories(first: 20, ownerAffiliations: OWNER, orderBy: {field: UPDATED_AT, direction: DESC})"`
	}
}

type gqlIssue struct {
	Number    int
	Title     string
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Author    struct{ Login string }
	Labels    struct {
		Nodes []struct {
			Name  string
			Color string
		}
	} `graphql:"labels(first: 10)"`
	Comments struct {
		TotalCount int `graphql:"totalCount"`
	}
}

type gqlRepo struct {
	Name           string
	Owner          struct{ Login string }
	Description    string
	IsPrivate      bool
	StargazerCount int
	ForkCount      int
	DefaultBranch  struct{ Name string }
	Languages      struct {
		Edges []struct {
			Node struct{ Name string }
		}
	} `graphql:"languages(first: 1, orderBy: {field: SIZE, direction: DESC})"`
}

// PRDetailQuery fetches a single PR with full details via GraphQL.
var prDetailQuery = struct {
	Repository *struct {
		PullRequest gqlPR `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}{}

// DashboardQuery fetches the viewer's dashboard data.
var dashboardQuery struct {
	Viewer struct {
		PullRequests struct {
			Nodes []gqlPR
		} `graphql:"pullRequests(first: 20, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC})"`
		Issues struct {
			Nodes []gqlIssue
		} `graphql:"issues(first: 20, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC})"`
		Repositories struct {
			Nodes []gqlRepo
		} `graphql:"repositories(first: 20, ownerAffiliations: OWNER, orderBy: {field: UPDATED_AT, direction: DESC})"`
	}
}

// --- Public methods ---

func (g *GraphQLClient) GetDashboard(ctx context.Context) (*models.Dashboard, error) {
	var query struct {
		Viewer struct {
			Login        string
			PullRequests struct {
				Nodes []gqlPR
			} `graphql:"pullRequests(first: 20, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC})"`
			Issues struct {
				Nodes []gqlIssue
			} `graphql:"issues(first: 20, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC})"`
			Repositories struct {
				Nodes []gqlRepo
			} `graphql:"repositories(first: 20, ownerAffiliations: OWNER, orderBy: {field: UPDATED_AT, direction: DESC})"`
		}
	}

	err := g.client.Query(ctx, &query, nil)
	if err != nil {
		return nil, fmt.Errorf("dashboard query: %w", err)
	}

	dash := &models.Dashboard{
		MyPRs:    make([]models.PullRequestSummary, 0, len(query.Viewer.PullRequests.Nodes)),
		MyIssues: make([]models.Issue, 0, len(query.Viewer.Issues.Nodes)),
		Repos:    make([]models.Repo, 0, len(query.Viewer.Repositories.Nodes)),
	}

	for _, pr := range query.Viewer.PullRequests.Nodes {
		dash.MyPRs = append(dash.MyPRs, convertGQLPRSummary(pr))
	}
	for _, i := range query.Viewer.Issues.Nodes {
		dash.MyIssues = append(dash.MyIssues, convertGQLIssue(i))
	}
	for _, r := range query.Viewer.Repositories.Nodes {
		dash.Repos = append(dash.Repos, convertGQLRepo(r))
	}

	return dash, nil
}

func (g *GraphQLClient) GetPullRequestGraphQL(ctx context.Context, owner, repo string, number int) (*models.PullRequestDetail, error) {
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"repo":   githubv4.String(repo),
		"number": githubv4.Int(number),
	}

	var query struct {
		Repository *struct {
			PullRequest gqlPR `graphql:"pullRequest(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	err := g.client.Query(ctx, &query, variables)
	if err != nil {
		return nil, fmt.Errorf("PR detail query for #%d: %w", number, err)
	}

	if query.Repository == nil {
		return nil, fmt.Errorf("repository %s/%s not found", owner, repo)
	}

	return convertGQLPRDetail(query.Repository.PullRequest), nil
}

// --- Converters ---

func convertGQLPRSummary(pr gqlPR) models.PullRequestSummary {
	state := models.PRStateOpen
	switch pr.State {
	case "CLOSED":
		state = models.PRStateClosed
	case "MERGED":
		state = models.PRStateMerged
	}
	if pr.IsDraft {
		state = models.PRStateDraft
	}

	labels := make([]string, 0, len(pr.Labels.Nodes))
	for _, l := range pr.Labels.Nodes {
		labels = append(labels, l.Name)
	}

	return models.PullRequestSummary{
		Number:    pr.Number,
		Title:     pr.Title,
		State:     state,
		Author:    pr.Author.Login,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
		IsDraft:   pr.IsDraft,
		Mergeable: pr.Mergeable,
		HeadRef:   pr.HeadRef.Name,
		BaseRef:   pr.BaseRef.Name,
		Labels:    labels,
	}
}

func convertGQLPRDetail(pr gqlPR) *models.PullRequestDetail {
	summary := convertGQLPRSummary(pr)

	reviews := make([]models.Review, 0, len(pr.Reviews.Nodes))
	for _, r := range pr.Reviews.Nodes {
		state := models.ReviewPending
		switch r.State {
		case "APPROVED":
			state = models.ReviewApproved
		case "CHANGES_REQUESTED":
			state = models.ReviewChangesRequested
		case "COMMENTED":
			state = models.ReviewCommented
		case "DISMISSED":
			state = models.ReviewDismissed
		}
		reviews = append(reviews, models.Review{
			Author:    r.Author.Login,
			State:     state,
			Body:      r.Body,
			Submitted: r.CreatedAt,
		})
	}

	checkRuns := make([]models.CheckRun, 0)
	if pr.StatusCheckRollup != nil {
		for _, c := range pr.StatusCheckRollup.Contexts {
			checkRuns = append(checkRuns, models.CheckRun{
				Name:       c.Name,
				State:      c.State,
				Conclusion: c.Conclusion,
				DetailsURL: c.DetailsURL,
			})
		}
	}

	files := make([]models.FileChange, 0, len(pr.Files.Nodes))
	for _, f := range pr.Files.Nodes {
		ft := models.FileModified
		switch f.ChangeType {
		case "ADDITION":
			ft = models.FileAdded
		case "DELETION":
			ft = models.FileDeleted
		case "RENAMED":
			ft = models.FileRenamed
		}
		files = append(files, models.FileChange{
			NewPath:   f.Path,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Type:      ft,
		})
	}

	detail := &models.PullRequestDetail{
		PullRequestSummary: summary,
		Body:               pr.Body,
		Additions:          pr.Additions,
		Deletions:          pr.Deletions,
		ChangedFiles:       pr.ChangedFiles,
		Reviews:            reviews,
		CheckRuns:          checkRuns,
		Files:              files,
		Comments:           pr.Comments.TotalCount,
	}

	return detail
}

func convertGQLIssue(i gqlIssue) models.Issue {
	state := models.IssueOpen
	if i.State == "CLOSED" {
		state = models.IssueClosed
	}

	labels := make([]models.Label, 0, len(i.Labels.Nodes))
	for _, l := range i.Labels.Nodes {
		labels = append(labels, models.Label{
			Name:  l.Name,
			Color: l.Color,
		})
	}

	return models.Issue{
		Number:    i.Number,
		Title:     i.Title,
		State:     state,
		Author:    i.Author.Login,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
		Labels:    labels,
		Comments:  i.Comments.TotalCount,
	}
}

func convertGQLRepo(r gqlRepo) models.Repo {
	lang := ""
	if len(r.Languages.Edges) > 0 {
		lang = r.Languages.Edges[0].Node.Name
	}
	return models.Repo{
		Owner:         r.Owner.Login,
		Name:          r.Name,
		FullName:      r.Owner.Login + "/" + r.Name,
		DefaultBranch: r.DefaultBranch.Name,
		Description:   r.Description,
		Language:      lang,
		IsPrivate:     r.IsPrivate,
		Stars:         r.StargazerCount,
		Forks:         r.ForkCount,
	}
}
