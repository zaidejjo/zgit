package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/github"
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <number>",
	Short: "Check out a pull request locally",
	Long: `Fetch pull/<number>/head and check out a local pr/<number> branch.

Works for fork PRs without adding extra remotes. Re-running after
new pushes force-updates the local branch.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		gh, err := ensureGitHub(ctx)
		if err != nil {
			return err
		}
		owner, repo, remote, err := getOwnerRepoAndRemote(ctx)
		if err != nil {
			return err
		}
		number := parseInt(args[0])
		if number <= 0 {
			return fmt.Errorf("invalid pull request number: %s", args[0])
		}
		if _, err := gh.GetPullRequest(ctx, owner, repo, number); err != nil {
			return fmt.Errorf("get PR #%d: %w", number, err)
		}
		refspec, branch := git.PRRefspec(number)
		if err := gitExec.FetchRefspec(ctx, remote, refspec); err != nil {
			return fmt.Errorf("fetch PR #%d: %w", number, err)
		}
		if err := gitExec.Checkout(ctx, branch); err != nil {
			return fmt.Errorf("checkout %s: %w", branch, err)
		}
		fmt.Fprintf(os.Stdout, "Checked out PR #%d on branch %s\n", number, branch)
		return nil
	},
}

var prChecksCmd = &cobra.Command{
	Use:   "checks <number>",
	Short: "Show CI checks and workflow runs for a pull request",
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
		if number <= 0 {
			return fmt.Errorf("invalid pull request number: %s", args[0])
		}
		detail, err := gh.GetPullRequest(ctx, owner, repo, number)
		if err != nil {
			return fmt.Errorf("get PR #%d: %w", number, err)
		}
		ref := detail.HeadSHA
		if ref == "" {
			ref = detail.HeadRef
		}
		checks, err := gh.ListCheckRunsForRef(ctx, owner, repo, ref)
		if err != nil {
			return fmt.Errorf("list checks: %w", err)
		}
		if len(checks) == 0 {
			fmt.Fprintln(os.Stdout, "No check runs found")
		} else {
			fmt.Fprintf(os.Stdout, "Checks for #%d (%s):\n", number, shortRef(ref))
			for _, c := range checks {
				fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", checkIcon(c.State, c.Conclusion), c.Name, checkState(c.State, c.Conclusion))
			}
		}

		// Workflow runs for the head commit, newest first.
		if detail.HeadSHA != "" {
			runs, err := gh.ListWorkflowRuns(ctx, owner, repo, github.RunsFilter{
				HeadSHA: detail.HeadSHA,
				Limit:   10,
			})
			if err == nil && len(runs) > 0 {
				fmt.Fprintln(os.Stdout, "\nWorkflow runs:")
				for _, r := range runs {
					fmt.Fprintf(os.Stdout, "  #%d %s [%s] %s\n", r.ID, r.WorkflowName, r.Status, r.Conclusion)
				}
			}
		}
		return nil
	},
}

// getOwnerRepoAndRemote is getOwnerRepo plus the remote name used.
func getOwnerRepoAndRemote(ctx context.Context) (owner, repo, remote string, err error) {
	remotes, err := gitExec.RemoteList(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("list remotes: %w", err)
	}
	for _, want := range []string{"origin", "upstream"} {
		for _, r := range remotes {
			if r.Name == want {
				owner, repo, err := parseGitHubRemote(r.URL)
				if err == nil {
					return owner, repo, r.Name, nil
				}
			}
		}
	}
	if len(remotes) > 0 {
		owner, repo, err := parseGitHubRemote(remotes[0].URL)
		if err == nil {
			return owner, repo, remotes[0].Name, nil
		}
		return "", "", "", err
	}
	return "", "", "", fmt.Errorf("no git remotes found")
}

func checkIcon(state, conclusion string) string {
	switch conclusion {
	case "SUCCESS", "NEUTRAL", "SKIPPED":
		return "✓"
	case "FAILURE", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED":
		return "✗"
	}
	if state == "IN_PROGRESS" || state == "QUEUED" || state == "PENDING" {
		return "…"
	}
	return "?"
}

func checkState(state, conclusion string) string {
	if conclusion != "" {
		return conclusion
	}
	return state
}

func shortRef(ref string) string {
	if len(ref) > 8 {
		return ref[:8]
	}
	return ref
}

func init() {
	prCmd.AddCommand(prCheckoutCmd)
	prCmd.AddCommand(prChecksCmd)
}
