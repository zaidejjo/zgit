package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v61/github"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *RESTClient) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := github.NewClient(nil)
	client.BaseURL = mustParseURL(server.URL + "/")
	return server, NewRESTClientWithClient(client)
}

func TestGetAuthenticatedUser(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(github.User{
			Login:     github.String("testuser"),
			Name:      github.String("Test User"),
			Bio:       github.String("A test user"),
			Followers: github.Int(42),
		})
	})
	defer server.Close()

	user, err := client.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Login != "testuser" {
		t.Errorf("Login = %q, want %q", user.Login, "testuser")
	}
	if user.Name != "Test User" {
		t.Errorf("Name = %q, want %q", user.Name, "Test User")
	}
	if user.Followers != 42 {
		t.Errorf("Followers = %d, want 42", user.Followers)
	}
}

func TestGetRepository(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/test-repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(github.Repository{
			FullName:        github.String("owner/test-repo"),
			Description:     github.String("A test repo"),
			DefaultBranch:   github.String("main"),
			StargazersCount: github.Int(100),
			ForksCount:      github.Int(20),
			OpenIssuesCount: github.Int(5),
			Private:         github.Bool(false),
		})
	})
	defer server.Close()

	repo, err := client.GetRepository(context.Background(), "owner", "test-repo")
	if err != nil {
		t.Fatal(err)
	}
	if repo.FullName != "owner/test-repo" {
		t.Errorf("FullName = %q, want %q", repo.FullName, "owner/test-repo")
	}
	if repo.Stars != 100 {
		t.Errorf("Stars = %d, want 100", repo.Stars)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want %q", repo.DefaultBranch, "main")
	}
}

func TestListIssues(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("First issue"),
				State:  github.String("open"),
				User:   &github.User{Login: github.String("alice")},
			},
			{
				Number: github.Int(2),
				Title:  github.String("Second issue"),
				State:  github.String("open"),
				User:   &github.User{Login: github.String("bob")},
			},
		})
	})
	defer server.Close()

	issues, err := client.ListIssues(context.Background(), "owner", "repo", IssuesFilter{State: "open"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Fatalf("len(issues) = %d, want 2", len(issues))
	}
	if issues[0].Number != 1 {
		t.Errorf("issues[0].Number = %d, want 1", issues[0].Number)
	}
	if issues[1].Author != "bob" {
		t.Errorf("issues[1].Author = %q, want %q", issues[1].Author, "bob")
	}
}

func TestListPullRequests(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]github.PullRequest{
			{
				Number: github.Int(42),
				Title:  github.String("Add feature"),
				State:  github.String("open"),
				User:   &github.User{Login: github.String("dev")},
				Head:   &github.PullRequestBranch{Ref: github.String("feature")},
				Base:   &github.PullRequestBranch{Ref: github.String("main")},
			},
		})
	})
	defer server.Close()

	prs, err := client.ListPullRequests(context.Background(), "owner", "repo", PRFilter{State: "open"})
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 {
		t.Fatalf("len(prs) = %d, want 1", len(prs))
	}
	if prs[0].Title != "Add feature" {
		t.Errorf("prs[0].Title = %q, want %q", prs[0].Title, "Add feature")
	}
}

func TestCreateIssue(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(github.Issue{
			Number: github.Int(10),
			Title:  github.String("New bug"),
			State:  github.String("open"),
			User:   &github.User{Login: github.String("testuser")},
		})
	})
	defer server.Close()

	issue, err := client.CreateIssue(context.Background(), "owner", "repo", IssueRequest{
		Title: "New bug",
		Body:  "Found a bug",
	})
	if err != nil {
		t.Fatal(err)
	}
	if issue.Number != 10 {
		t.Errorf("Number = %d, want 10", issue.Number)
	}
	if issue.Title != "New bug" {
		t.Errorf("Title = %q, want %q", issue.Title, "New bug")
	}
}

func TestListWorkflows(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"workflows": []github.Workflow{
				{
					ID:    github.Int64(100),
					Name:  github.String("CI"),
					Path:  github.String(".github/workflows/ci.yml"),
					State: github.String("active"),
				},
			},
		})
	})
	defer server.Close()

	workflows, err := client.ListWorkflows(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) != 1 {
		t.Fatalf("len(workflows) = %d, want 1", len(workflows))
	}
	if workflows[0].Name != "CI" {
		t.Errorf("Name = %q, want %q", workflows[0].Name, "CI")
	}
}

func TestListWorkflowRuns(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"workflow_runs": []github.WorkflowRun{
				{
					ID:         github.Int64(999),
					Name:       github.String("test-run"),
					Status:     github.String("completed"),
					Conclusion: github.String("success"),
					HeadBranch: github.String("main"),
					Event:      github.String("push"),
				},
			},
		})
	})
	defer server.Close()

	runs, err := client.ListWorkflowRuns(context.Background(), "owner", "repo", RunsFilter{Branch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(runs))
	}
	if runs[0].WorkflowName != "test-run" {
		t.Errorf("WorkflowName = %q, want %q", runs[0].WorkflowName, "test-run")
	}
}

func TestTestToken(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(github.User{
			Login: github.String("testuser"),
		})
	})
	defer server.Close()

	if err := client.TestToken(context.Background()); err != nil {
		t.Errorf("TestToken() = %v, want nil", err)
	}
}

func TestRESTListRepositories(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]github.Repository{
			{
				FullName: github.String("user/repo1"),
				Name:     github.String("repo1"),
			},
			{
				FullName: github.String("user/repo2"),
				Name:     github.String("repo2"),
			},
		})
	})
	defer server.Close()

	repos, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("len(repos) = %d, want 2", len(repos))
	}
}

func TestAPIError(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})
	defer server.Close()

	_, err := client.GetRepository(context.Background(), "nonexistent", "repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCloseIssue(t *testing.T) {
	server, client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(github.Issue{
			Number: github.Int(5),
			State:  github.String("closed"),
		})
	})
	defer server.Close()

	if err := client.CloseIssue(context.Background(), "owner", "repo", 5); err != nil {
		t.Errorf("CloseIssue() = %v, want nil", err)
	}
}
