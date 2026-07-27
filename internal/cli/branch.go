package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "List, create, or delete branches",
	Long:  `Manage Git branches: list all branches, create new ones, or delete existing ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		branches, err := gitExec.Branches(ctx)
		if err != nil {
			return fmt.Errorf("list branches: %w", err)
		}

		if len(branches) == 0 {
			fmt.Fprintln(os.Stdout, "No branches found")
			return nil
		}

		for _, b := range branches {
			marker := "  "
			if b.IsHead {
				marker = "* "
			}

			upstreamInfo := ""
			if b.Upstream != "" {
				upstreamInfo = fmt.Sprintf(" [%s", b.Upstream)
				if b.Ahead > 0 || b.Behind > 0 {
					if b.Ahead > 0 {
						upstreamInfo += fmt.Sprintf(" ahead %d", b.Ahead)
					}
					if b.Behind > 0 {
						upstreamInfo += fmt.Sprintf(" behind %d", b.Behind)
					}
				}
				upstreamInfo += "]"
			}

			fmt.Fprintf(os.Stdout, "%s%s%s\n", marker, b.Name, upstreamInfo)
		}

		return nil
	},
}
