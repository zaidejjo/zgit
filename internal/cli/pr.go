package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/github"
)

var prCmd = &cobra.Command{
	Use:     "pr",
	Aliases: []string{"pull-request"},
	Short:   "Manage pull requests",
	Long:    `List, view, and manage GitHub pull requests for the current repository.`,
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long:  `Display pull requests for the current repository with status indicators.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		gh, err := ensureGitHub(ctx)
		if err != nil {
			return err
		}

		owner, repo, err := getOwnerRepo(ctx)
		if err != nil {
			return fmt.Errorf("determine repo: %w\n(Hint: run 'zgit auth login' first, then navigate to a git repo with GitHub remote)", err)
		}

		state, _ := cmd.Flags().GetString("state")
		limit, _ := cmd.Flags().GetInt("limit")

		prs, err := gh.ListPullRequests(ctx, owner, repo, github.PRFilter{
			State: state,
			Limit: limit,
			Sort:  "updated",
		})
		if err != nil {
			return fmt.Errorf("list PRs: %w", err)
		}

		if len(prs) == 0 {
			fmt.Fprintf(os.Stdout, "No %s pull requests in %s/%s\n", state, owner, repo)
			return nil
		}

		fmt.Fprintf(os.Stdout, "Pull requests for %s/%s (%s):\n", owner, repo, state)
		for _, pr := range prs {
			draftMark := " "
			if pr.IsDraft {
				draftMark = "D"
			}
			mergeable := prStatus(pr.Mergeable)
			fmt.Fprintf(os.Stdout, "  #%-5d %s %s %s\n", pr.Number, prStatusEmoji(pr.State), pr.Title, draftMark)
			fmt.Fprintf(os.Stdout, "       %s  by %s  %s\n", mergeable, pr.Author, timeSince(pr.UpdatedAt))
		}
		return nil
	},
}

var prViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View pull request details",
	Long:  `Show detailed information about a specific pull request including status checks and reviews.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		gh, err := ensureGitHub(ctx)
		if err != nil {
			return err
		}

		owner, repo, err := getOwnerRepo(ctx)
		if err != nil {
			return err
		}

		number := parseInt(args[0])
		detail, err := gh.GetPullRequest(ctx, owner, repo, number)
		if err != nil {
			return fmt.Errorf("get PR #%d: %w", number, err)
		}

		fmt.Fprintf(os.Stdout, "#%d %s\n", detail.Number, detail.Title)
		fmt.Fprintf(os.Stdout, "State: %s  ", detail.State)
		if detail.IsDraft {
			fmt.Fprintf(os.Stdout, "[DRAFT] ")
		}
		fmt.Fprintf(os.Stdout, "Mergeable: %s\n", detail.Mergeable)
		fmt.Fprintf(os.Stdout, "Author: %s\n", detail.Author)
		fmt.Fprintf(os.Stdout, "Branch: %s -> %s\n", detail.HeadRef, detail.BaseRef)
		fmt.Fprintf(os.Stdout, "Updated: %s\n", timeSince(detail.UpdatedAt))
		fmt.Fprintf(os.Stdout, "Changes: +%d/-%d in %d files\n", detail.Additions, detail.Deletions, detail.ChangedFiles)

		// Status checks
		if len(detail.CheckRuns) > 0 {
			fmt.Fprintf(os.Stdout, "\nChecks:\n")
			for _, c := range detail.CheckRuns {
				icon := "✓"
				if c.Conclusion == "FAILURE" || c.Conclusion == "CANCELLED" {
					icon = "✗"
				} else if c.State == "IN_PROGRESS" || c.State == "PENDING" {
					icon = "…"
				}
				fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", icon, c.Name, c.Conclusion)
			}
		}

		// Reviews
		if len(detail.Reviews) > 0 {
			fmt.Fprintf(os.Stdout, "\nReviews:\n")
			for _, r := range detail.Reviews {
				icon := "💬"
				switch r.State {
				case "APPROVED":
					icon = "✓"
				case "CHANGES_REQUESTED":
					icon = "✗"
				}
				fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", icon, r.Author, r.State)
			}
		}

		// Body preview
		if detail.Body != "" {
			fmt.Fprintf(os.Stdout, "\nDescription:\n")
			maxBody := detail.Body
			if len(maxBody) > 500 {
				maxBody = maxBody[:500] + "…"
			}
			fmt.Fprintf(os.Stdout, "%s\n", maxBody)
		}

		// Files changed
		if len(detail.Files) > 0 {
			fmt.Fprintf(os.Stdout, "\nFiles changed (%d):\n", len(detail.Files))
			for _, f := range detail.Files {
				fmt.Fprintf(os.Stdout, "  %s +%d/-%d\n", f.NewPath, f.Additions, f.Deletions)
			}
		}

		return nil
	},
}

func init() {
	prListCmd.Flags().StringP("state", "s", "open", "filter by state (open, closed, all)")
	prListCmd.Flags().IntP("limit", "n", 20, "max number of PRs")
	prViewCmd.Flags().Bool("checks", true, "show status checks")
	prViewCmd.Flags().Bool("reviews", true, "show reviews")

	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prViewCmd)
}

func prStatusEmoji(state interface{}) string {
	s, ok := state.(string)
	if !ok {
		return "•"
	}
	switch s {
	case "OPEN", "open":
		return "🟢"
	case "MERGED", "merged":
		return "🟣"
	case "CLOSED", "closed":
		return "🔴"
	case "DRAFT", "draft":
		return "⚪"
	}
	return "•"
}

func prStatus(s string) string {
	switch s {
	case "MERGEABLE":
		return "✓"
	case "CONFLICTING":
		return "✗"
	case "UNKNOWN":
		return "?"
	default:
		return " "
	}
}

func timeSince(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// getOwnerRepo extracts owner/repo from git remotes or flags.
func getOwnerRepo(ctx context.Context) (string, string, error) {
	// Try to get from git remote
	remotes, err := gitExec.RemoteList(ctx)
	if err != nil {
		return "", "", fmt.Errorf("list remotes: %w", err)
	}
	for _, r := range remotes {
		if r.Name == "origin" || r.Name == "upstream" {
			owner, repo, err := parseGitHubRemote(r.URL)
			if err == nil {
				return owner, repo, nil
			}
		}
	}
	if len(remotes) > 0 {
		return parseGitHubRemote(remotes[0].URL)
	}
	return "", "", fmt.Errorf("no git remotes found")
}

func parseGitHubRemote(url string) (string, string, error) {
	url = strings.TrimSuffix(url, ".git")
	var path string
	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		path = parts[len(parts)-1]
	} else if strings.Contains(url, "github.com:") {
		parts := strings.Split(url, "github.com:")
		path = parts[len(parts)-1]
	} else {
		return "", "", fmt.Errorf("not a GitHub remote")
	}
	segments := strings.Split(path, "/")
	if len(segments) >= 2 {
		return segments[0], segments[1], nil
	}
	return "", "", fmt.Errorf("cannot parse remote: %s", url)
}

// ensureGitHub returns the GitHub client from the engine, or errors if not authenticated.
func ensureGitHub(ctx context.Context) (github.GitHubClient, error) {
	token := cfg.GetString("github.token")
	if token == "" {
		return nil, fmt.Errorf("not authenticated with GitHub\nRun 'zgit auth login' first")
	}
	client, err := github.NewCombinedClient(token)
	if err != nil {
		return nil, fmt.Errorf("init github client: %w", err)
	}
	return client, nil
}
