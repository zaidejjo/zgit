package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

var blameOpts struct {
	ref   string
	lines string
}

var blameCmd = &cobra.Command{
	Use:   "blame <file>",
	Short: "Show per-line commit attribution for a file",
	Long: `Attribute each line of <file> to the commit that last modified it.

Examples:
  zgit blame src/main.go
  zgit blame src/main.go --ref HEAD~3
  zgit blame src/main.go --lines 10,20`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		lines, err := gitExec.Blame(ctx, args[0], models.BlameOptions{
			Ref:   blameOpts.ref,
			Lines: blameOpts.lines,
		})
		if err != nil {
			return fmt.Errorf("blame %s: %w", args[0], err)
		}
		if len(lines) == 0 {
			fmt.Fprintln(os.Stdout, "No blame output (empty file?)")
			return nil
		}
		for _, l := range lines {
			hash := l.Hash
			if len(hash) > 8 {
				hash = hash[:8]
			}
			if l.IsUncommitted {
				hash = "00000000"
			}
			fmt.Fprintf(os.Stdout, "%s %-12.12s %4d  %s\n", hash, l.Author, l.LineNo, l.Content)
		}
		return nil
	},
}

func init() {
	blameCmd.Flags().StringVar(&blameOpts.ref, "ref", "", "revision to blame (default: working tree)")
	blameCmd.Flags().StringVarP(&blameOpts.lines, "lines", "L", "", "line range, e.g. \"10,20\"")
	rootCmd.AddCommand(blameCmd)
}
