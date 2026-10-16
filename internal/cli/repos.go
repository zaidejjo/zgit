package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

var reposFetch bool
var reposPrune bool
var reposRemove string

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Overview of recent and favorite repositories",
	Long: `Show branch, cleanliness and ahead/behind for recent and favorite repos.

Use --fetch to fetch all remotes first. Star favorites in the
desktop dashboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		paths := engine.DashboardRepos()
		if len(paths) == 0 {
			fmt.Fprintln(os.Stdout, "No recent repositories — open one with 'zgit -C <path>' first")
			return nil
		}
		if reposRemove != "" {
			if err := engine.RemoveRepo(reposRemove); err != nil {
				return fmt.Errorf("remove repo: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Forgot %s (files on disk untouched)\n", reposRemove)
			paths = engine.DashboardRepos()
		}
		if reposPrune {
			n, err := engine.PruneDeadRepos()
			if err != nil {
				return fmt.Errorf("prune dead repos: %w", err)
			}
			if n == 1 {
				fmt.Fprintln(os.Stdout, "Pruned 1 dead entry")
			} else if n > 1 {
				fmt.Fprintf(os.Stdout, "Pruned %d dead entries\n", n)
			}
			paths = engine.DashboardRepos()
		}
		for _, p := range paths {
			if reposFetch {
				if err := engine.FetchRepo(p); err != nil {
					fmt.Fprintf(os.Stdout, "%-40s fetch failed: %v\n", p, err)
					continue
				}
			}
			ov := engine.RepoOverview(p, true)
			fmt.Fprintln(os.Stdout, formatOverview(ov))
		}
		return nil
	},
}

func formatOverview(ov *models.RepoOverview) string {
	var sb strings.Builder
	mark := " "
	if ov.Favorite {
		mark = "*"
	}
	fmt.Fprintf(&sb, "%s %-30s", mark, ov.Name)
	if ov.Error != "" {
		fmt.Fprintf(&sb, "  [unreadable: %s]", shortErr(ov.Error))
		return sb.String()
	}
	fmt.Fprintf(&sb, "  (%s)", ov.Branch)
	if ov.IsClean {
		sb.WriteString("  clean")
	} else {
		fmt.Fprintf(&sb, "  %d dirty", ov.DirtyCount)
	}
	if ov.Ahead > 0 || ov.Behind > 0 {
		fmt.Fprintf(&sb, "  ↑%d ↓%d", ov.Ahead, ov.Behind)
	}
	if ov.LastMessage != "" {
		msg := ov.LastMessage
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}
		fmt.Fprintf(&sb, "  %s", msg)
	}
	return sb.String()
}

func shortErr(err string) string {
	lower := strings.ToLower(err)
	switch {
	case strings.Contains(lower, "not a git repository"):
		return "not a git repository"
	case strings.Contains(lower, "no such file"):
		return "path gone"
	}
	line := strings.TrimSpace(strings.SplitN(err, "\n", 2)[0])
	if len(line) > 60 {
		return line[:57] + "..."
	}
	return line
}

func init() {
	reposCmd.Flags().BoolVar(&reposFetch, "fetch", false, "fetch all remotes before showing status")
	reposCmd.Flags().BoolVar(&reposPrune, "prune", false, "drop recent entries whose paths no longer exist")
	reposCmd.Flags().StringVar(&reposRemove, "remove", "", "forget a repo in the app (files on disk untouched)")
	rootCmd.AddCommand(reposCmd)
	reposCmd.AddCommand(reposAddCmd)
	reposCmd.AddCommand(reposScanCmd)
}

var reposAddCmd = &cobra.Command{
	Use:   "add <path>...",
	Short: "Add repositories to the app without opening them",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		if n := engine.TrackRepos(args); n == 1 {
			fmt.Fprintln(os.Stdout, "Added 1 repo")
		} else {
			fmt.Fprintf(os.Stdout, "Added %d repos\n", n)
		}
		return nil
	},
}

var reposScanCmd = &cobra.Command{
	Use:   "scan <dir>",
	Short: "Find git repositories under a directory and add them all",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		found, err := engine.ScanForRepos(args[0], 4)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		paths := make([]string, 0, len(found))
		for _, r := range found {
			mark := ""
			if r.Tracked {
				mark = "(already added)"
			} else {
				paths = append(paths, r.Path)
			}
			fmt.Fprintf(os.Stdout, "%-50s %s\n", r.Path, mark)
		}
		if n := engine.TrackRepos(paths); n == 1 {
			fmt.Fprintln(os.Stdout, "Added 1 repo")
		} else {
			fmt.Fprintf(os.Stdout, "Added %d repos\n", n)
		}
		return nil
	},
}
