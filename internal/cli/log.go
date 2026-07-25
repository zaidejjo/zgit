package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/git"
)

var logOpts struct {
	count    int
	branch   string
	author   string
	since    string
	until    string
	all      bool
	merges   bool
	noMerges bool
	file     string
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit log",
	Long:  `Display the commit history in a readable format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		opts := git.LogOptions{
			Count:    logOpts.count,
			Branch:   logOpts.branch,
			Author:   logOpts.author,
			Since:    logOpts.since,
			Until:    logOpts.until,
			All:      logOpts.all,
			Merges:   logOpts.merges,
			NoMerges: logOpts.noMerges,
			File:     logOpts.file,
		}

		commits, err := gitExec.Log(ctx, opts)
		if err != nil {
			return fmt.Errorf("get log: %w", err)
		}

		if len(commits) == 0 {
			fmt.Fprintln(os.Stdout, "No commits found")
			return nil
		}

		for _, c := range commits {
			shortHash := c.Hash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}

			refInfo := ""
			if c.RefNames != "" {
				refInfo = " " + c.RefNames
			}

			fmt.Fprintf(os.Stdout, "%s%s\n", shortHash, refInfo)
			fmt.Fprintf(os.Stdout, "  Author: %s <%s>\n", c.Author, c.Email)
			fmt.Fprintf(os.Stdout, "  Date:   %s\n", c.Timestamp.Format(time.RFC1123))
			fmt.Fprintf(os.Stdout, "\n  %s\n\n", c.Message)
		}

		return nil
	},
}

func init() {
	logCmd.Flags().IntVarP(&logOpts.count, "count", "n", 0, "limit number of commits")
	logCmd.Flags().StringVarP(&logOpts.branch, "branch", "b", "", "branch to show log for")
	logCmd.Flags().StringVarP(&logOpts.author, "author", "A", "", "filter by author")
	logCmd.Flags().StringVarP(&logOpts.since, "since", "S", "", "show commits after date")
	logCmd.Flags().StringVarP(&logOpts.until, "until", "U", "", "show commits before date")
	logCmd.Flags().BoolVar(&logOpts.all, "all", false, "show all branches")
	logCmd.Flags().BoolVar(&logOpts.merges, "merges", false, "show only merges")
	logCmd.Flags().BoolVar(&logOpts.noMerges, "no-merges", false, "exclude merges")
	logCmd.Flags().StringVarP(&logOpts.file, "file", "f", "", "show log for file")
}
