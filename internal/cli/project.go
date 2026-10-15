package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

var (
	projectColor string
	cleanupBase  string
	cleanupYes   bool
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Group repositories into projects",
	Long: `Create named projects holding multiple repositories, and run
bulk operations across members.

Examples:
  zgit project list
  zgit project create work --color "#89b4fa"
  zgit project add work ~/Projects/Apps/zgit
  zgit project fetch work`,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		projects := engine.GetProjects()
		if len(projects) == 0 {
			fmt.Fprintln(os.Stdout, "No projects yet — create one with 'zgit project create <name>'")
			return nil
		}
		for _, p := range projects {
			fmt.Fprintf(os.Stdout, "%s  %s (%d repos)\n", p.ID[:8], p.Name, len(p.Repos))
			for _, r := range p.Repos {
				fmt.Fprintf(os.Stdout, "    %s\n", r)
			}
		}
		return nil
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		id, err := engine.CreateProject(args[0], projectColor)
		if err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Created project %q (%s)\n", args[0], id[:8])
		return nil
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a project (repos stay)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		p, err := resolveProject(engine, args[0])
		if err != nil {
			return err
		}
		if err := engine.DeleteProject(p.ID); err != nil {
			return fmt.Errorf("delete project: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Deleted project %q\n", p.Name)
		return nil
	},
}

var projectAddCmd = &cobra.Command{
	Use:   "add <name|id> <path>",
	Short: "Add a repo to a project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		p, err := resolveProject(engine, args[0])
		if err != nil {
			return err
		}
		if err := engine.AddProjectRepo(p.ID, args[1]); err != nil {
			return fmt.Errorf("add repo: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Added %s to %q\n", args[1], p.Name)
		return nil
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:   "remove <name|id> <path>",
	Short: "Remove a repo from a project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		p, err := resolveProject(engine, args[0])
		if err != nil {
			return err
		}
		if err := engine.RemoveProjectRepo(p.ID, args[1]); err != nil {
			return fmt.Errorf("remove repo: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Removed %s from %q\n", args[1], p.Name)
		return nil
	},
}

var projectFetchCmd = &cobra.Command{
	Use:   "fetch <name|id>",
	Short: "Fetch all member repos",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		p, err := resolveProject(engine, args[0])
		if err != nil {
			return err
		}
		n, err := engine.FetchProject(p.ID)
		if err != nil {
			return fmt.Errorf("fetch project: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Fetched %d repos in %q\n", n, p.Name)
		return nil
	},
}

// resolveProject matches by exact name (case-insensitive) or id prefix.
func resolveProject(engine *core.Engine, ref string) (*models.Project, error) {
	projects := engine.GetProjects()
	for i := range projects {
		if strings.EqualFold(projects[i].Name, ref) {
			return &projects[i], nil
		}
	}
	var prefix []*models.Project
	for i := range projects {
		if strings.HasPrefix(projects[i].ID, strings.ToLower(ref)) {
			prefix = append(prefix, &projects[i])
		}
	}
	if len(prefix) == 1 {
		return prefix[0], nil
	}
	if len(prefix) > 1 {
		return nil, fmt.Errorf("ambiguous project ref %q", ref)
	}
	return nil, fmt.Errorf("project not found: %s", ref)
}

func init() {
	projectCreateCmd.Flags().StringVar(&projectColor, "color", "", "accent color, e.g. \"#89b4fa\"")
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectFetchCmd)
	projectCleanupCmd.Flags().StringVar(&cleanupBase, "base", "", "merge base (default: each repo's current branch)")
	projectCleanupCmd.Flags().BoolVar(&cleanupYes, "yes", false, "skip confirmation")
	projectCmd.AddCommand(projectCleanupCmd)
	rootCmd.AddCommand(projectCmd)
}

var projectCleanupCmd = &cobra.Command{
	Use:   "cleanup <name|id>",
	Short: "Preview and delete branches merged into base across members",
	Long: `Scan every member repo for branches fully merged into --base
(default: each repo's current branch). Protected branches (main,
master, dev, base, checked-out) are withheld, never deleted.
Without --yes, prints the plan and asks once.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := core.New(core.DefaultEngineOptions())
		if err != nil {
			return fmt.Errorf("init engine: %w", err)
		}
		p, err := resolveProject(engine, args[0])
		if err != nil {
			return err
		}
		candidates, withheld, err := engine.ProjectCleanupScan(p.ID, cleanupBase)
		if err != nil {
			return fmt.Errorf("scan project: %w", err)
		}
		if len(candidates) == 0 {
			fmt.Fprintln(os.Stdout, "Nothing to clean — no merged branches found")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Will delete %d branches in %q:\n", len(candidates), p.Name)
		for _, c := range candidates {
			fmt.Fprintf(os.Stdout, "  %-20s %s\n", c.RepoName+"/"+c.Branch, c.LastMessage)
		}
		if len(withheld) > 0 {
			fmt.Fprintf(os.Stdout, "(withheld %d protected/checked-out branches)\n", len(withheld))
		}
		if !cleanupYes {
			fmt.Fprint(os.Stdout, "Delete them? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Fprintln(os.Stdout, "Aborted")
				return nil
			}
		}
		failed := 0
		for _, c := range candidates {
			if err := engine.DeleteRepoBranch(c.RepoPath, c.Branch); err != nil {
				fmt.Fprintf(os.Stdout, "  ✗ %s: %v\n", c.RepoName+"/"+c.Branch, err)
				failed++
			}
		}
		deleted := len(candidates) - failed
		if failed == 0 {
			fmt.Fprintf(os.Stdout, "Deleted %d branches\n", deleted)
		} else {
			fmt.Fprintf(os.Stdout, "Deleted %d branches (%d failed)\n", deleted, failed)
		}
		return nil
	},
}
