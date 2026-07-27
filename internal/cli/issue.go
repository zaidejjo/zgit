package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/github"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
	Long:  `List, view, and create GitHub issues for the current repository.`,
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long:  `Display GitHub issues for the current repository.`,
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

		state, _ := cmd.Flags().GetString("state")
		label, _ := cmd.Flags().GetString("label")
		limit, _ := cmd.Flags().GetInt("limit")

		issues, err := gh.ListIssues(ctx, owner, repo, github.IssuesFilter{
			State: state,
			Label: label,
			Sort:  "updated",
			Limit: limit,
		})
		if err != nil {
			return fmt.Errorf("list issues: %w", err)
		}

		if len(issues) == 0 {
			fmt.Fprintf(os.Stdout, "No %s issues in %s/%s\n", state, owner, repo)
			return nil
		}

		fmt.Fprintf(os.Stdout, "Issues for %s/%s (%s):\n", owner, repo, state)
		for _, i := range issues {
			labels := ""
			if len(i.Labels) > 0 {
				for _, l := range i.Labels {
					labels += " [" + l.Name + "]"
				}
			}
			fmt.Fprintf(os.Stdout, "  #%-5d %s%s\n", i.Number, i.Title, labels)
			fmt.Fprintf(os.Stdout, "       by %s  %s\n", i.Author, timeSince(i.UpdatedAt))
		}
		return nil
	},
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an issue",
	Long:  `Create a new GitHub issue with title, body, labels, and assignees.`,
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

		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		labels, _ := cmd.Flags().GetStringSlice("label")

		if title == "" {
			return fmt.Errorf("title is required (use --title)")
		}

		created, err := gh.CreateIssue(ctx, owner, repo, github.IssueRequest{
			Title:  title,
			Body:   body,
			Labels: labels,
		})
		if err != nil {
			return fmt.Errorf("create issue: %w", err)
		}

		fmt.Fprintf(os.Stdout, "✓ Created issue #%d: %s\n", created.Number, created.Title)
		return nil
	},
}

func init() {
	issueListCmd.Flags().StringP("state", "s", "open", "filter by state (open, closed, all)")
	issueListCmd.Flags().StringP("label", "l", "", "filter by label")
	issueListCmd.Flags().IntP("limit", "n", 30, "max number of issues")

	issueCreateCmd.Flags().StringP("title", "t", "", "issue title (required)")
	issueCreateCmd.Flags().StringP("body", "b", "", "issue body")
	issueCreateCmd.Flags().StringSlice("label", nil, "labels to apply")

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueCreateCmd)
}
