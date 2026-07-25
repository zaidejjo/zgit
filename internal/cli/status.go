package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

func statusMark(t models.StatusType) string {
	switch t {
	case models.StatusAdded:
		return "A"
	case models.StatusModified:
		return "M"
	case models.StatusDeleted:
		return "D"
	case models.StatusRenamed:
		return "R"
	case models.StatusCopied:
		return "C"
	case models.StatusUntracked:
		return "?"
	default:
		return " "
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show working tree status",
	Long:  `Display the current repository status including staged, unstaged, and untracked files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		status, err := gitExec.Status(ctx)
		if err != nil {
			return fmt.Errorf("get status: %w", err)
		}

		fmt.Fprintf(os.Stdout, "On branch %s\n", status.Branch)
		if status.Upstream != "" {
			fmt.Fprintf(os.Stdout, "Upstream: %s", status.Upstream)
			if status.Ahead > 0 || status.Behind > 0 {
				fmt.Fprintf(os.Stdout, " [")
				if status.Ahead > 0 {
					fmt.Fprintf(os.Stdout, "ahead %d", status.Ahead)
				}
				if status.Ahead > 0 && status.Behind > 0 {
					fmt.Fprintf(os.Stdout, ", ")
				}
				if status.Behind > 0 {
					fmt.Fprintf(os.Stdout, "behind %d", status.Behind)
				}
				fmt.Fprintf(os.Stdout, "]")
			}
			fmt.Fprintln(os.Stdout)
		}

		if status.IsClean {
			fmt.Fprintln(os.Stdout, "\nNothing to commit, working tree clean")
			return nil
		}

		if status.IsMerging {
			fmt.Fprintln(os.Stdout, "\n(Merging in progress)")
		}
		if status.IsRebasing {
			fmt.Fprintln(os.Stdout, "\n(Rebasing in progress)")
		}

		// Staged files
		staged := status.StagedFiles()
		if len(staged) > 0 {
			fmt.Fprintf(os.Stdout, "\nChanges staged for commit:\n")
			for _, f := range staged {
				fmt.Fprintf(os.Stdout, "  %s %s\n", statusMark(f.Staged), f.Path)
			}
		}

		// Unstaged files
		unstaged := status.UnstagedFiles()
		if len(unstaged) > 0 {
			fmt.Fprintf(os.Stdout, "\nChanges not staged for commit:\n")
			for _, f := range unstaged {
				fmt.Fprintf(os.Stdout, "  %s %s\n", statusMark(f.Unstaged), f.Path)
			}
		}

		// Untracked files
		untracked := status.UntrackedFiles()
		if len(untracked) > 0 {
			fmt.Fprintf(os.Stdout, "\nUntracked files:\n")
			for _, f := range untracked {
				fmt.Fprintf(os.Stdout, "  %s %s\n", statusMark(f.Staged), f.Path)
			}
		}

		return nil
	},
}
