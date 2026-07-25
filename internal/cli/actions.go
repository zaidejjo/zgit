package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/github"
)

var actionsCmd = &cobra.Command{
	Use:     "actions",
	Aliases: []string{"ci", "workflow"},
	Short:   "Manage GitHub Actions",
	Long:    `List, view, and manage GitHub Actions workflow runs for the current repository.`,
}

var actionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflow runs",
	Long:  `Display recent GitHub Actions workflow runs for the current repository.`,
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

		branch, _ := cmd.Flags().GetString("branch")
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")

		runs, err := gh.ListWorkflowRuns(ctx, owner, repo, github.RunsFilter{
			Branch: branch,
			Status: status,
			Limit:  limit,
		})
		if err != nil {
			return fmt.Errorf("list workflow runs: %w", err)
		}

		if len(runs) == 0 {
			fmt.Fprintf(os.Stdout, "No workflow runs found in %s/%s\n", owner, repo)
			return nil
		}

		fmt.Fprintf(os.Stdout, "Workflow runs for %s/%s:\n", owner, repo)
		for _, r := range runs {
			icon := runStatusIcon(r.Status)
			fmt.Fprintf(os.Stdout, "  %s #%-5d %s\n", icon, r.RunNumber, r.WorkflowName)
			fmt.Fprintf(os.Stdout, "       %s → %s  %s\n", r.Event, r.Branch, timeSince(r.UpdatedAt))
		}
		return nil
	},
}

var actionsViewCmd = &cobra.Command{
	Use:   "view <run-id>",
	Short: "View workflow run details",
	Long:  `Show detailed information about a specific workflow run including jobs and their status.`,
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

		runID := parseInt64(args[0])

		run, err := gh.GetWorkflowRun(ctx, owner, repo, runID)
		if err != nil {
			return fmt.Errorf("get workflow run: %w", err)
		}

		fmt.Fprintf(os.Stdout, "Run #%d: %s\n", run.RunNumber, run.WorkflowName)
		fmt.Fprintf(os.Stdout, "Status: %s\n", run.Status)
		fmt.Fprintf(os.Stdout, "Event: %s  Branch: %s\n", run.Event, run.Branch)
		fmt.Fprintf(os.Stdout, "Commit: %s\n", run.HeadSHA[:8])
		fmt.Fprintf(os.Stdout, "%s\n", timeSince(run.CreatedAt))

		// Jobs
		jobs, err := gh.ListWorkflowJobs(ctx, owner, repo, runID)
		if err == nil {
			fmt.Fprintf(os.Stdout, "\nJobs:\n")
			for _, j := range jobs {
				icon := runStatusIconFromString(j.Status)
				fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", icon, j.Name, j.Conclusion)
			}
		}

		return nil
	},
}

var actionsReRunCmd = &cobra.Command{
	Use:   "rerun <run-id>",
	Short: "Re-run a workflow",
	Long:  `Trigger a re-run of a completed workflow run.`,
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

		runID := parseInt64(args[0])

		if err := gh.ReRunWorkflow(ctx, owner, repo, runID); err != nil {
			return fmt.Errorf("rerun workflow: %w", err)
		}

		fmt.Fprintf(os.Stdout, "✓ Re-run triggered for run #%d\n", runID)
		return nil
	},
}

var actionsCancelCmd = &cobra.Command{
	Use:   "cancel <run-id>",
	Short: "Cancel a workflow run",
	Long:  `Cancel a currently running workflow.`,
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

		runID := parseInt64(args[0])

		if err := gh.CancelWorkflowRun(ctx, owner, repo, runID); err != nil {
			return fmt.Errorf("cancel workflow: %w", err)
		}

		fmt.Fprintf(os.Stdout, "✓ Cancel requested for run #%d\n", runID)
		return nil
	},
}

func init() {
	actionsListCmd.Flags().StringP("branch", "b", "", "filter by branch")
	actionsListCmd.Flags().StringP("status", "s", "", "filter by status (completed, in_progress, queued)")
	actionsListCmd.Flags().IntP("limit", "n", 20, "max number of runs")

	actionsCmd.AddCommand(actionsListCmd)
	actionsCmd.AddCommand(actionsViewCmd)
	actionsCmd.AddCommand(actionsReRunCmd)
	actionsCmd.AddCommand(actionsCancelCmd)
}

func runStatusIcon(s interface{}) string {
	str, ok := s.(string)
	if !ok {
		return "•"
	}
	switch str {
	case "success", "SUCCESS":
		return "✓"
	case "failure", "FAILURE":
		return "✗"
	case "in_progress", "IN_PROGRESS":
		return "…"
	case "queued", "QUEUED":
		return "○"
	case "cancelled", "CANCELLED":
		return "⊘"
	case "skipped", "SKIPPED":
		return "⤵"
	case "stale", "STALE":
		return "⚠"
	default:
		return "•"
	}
}

func runStatusIconFromString(s string) string {
	return runStatusIcon(s)
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
