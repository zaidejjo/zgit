// Package cli implements the zgit command-line interface using cobra.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/git"
)

var (
	repoPath string
	gitExec  *git.NativeExec
)

// rootCmd is the base command for zgit.
var rootCmd = &cobra.Command{
	Use:   "zgit",
	Short: "A modern, fast Git & GitHub client",
	Long: `zgit combines local Git operations with GitHub CLI/API features
(PRs, Issues, Actions, Reviews) into a clean, non-cluttered interface.

Supports both CLI commands and an interactive TUI.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize git backend on any command except help/completion
		if cmd.Use == "help" || cmd.Use == "completion" {
			return nil
		}
		var err error
		gitExec, err = git.NewNativeExec("")
		if err != nil {
			return fmt.Errorf("initialize git: %w", err)
		}
		path := repoPath
		if path == "" {
			path = "."
		}
		if err := gitExec.Open(path); err != nil {
			return fmt.Errorf("open repo: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if gitExec != nil {
			return gitExec.Close()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show help if no subcommand
		return cmd.Help()
	},
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "C", "", "path to git repository (default: current directory)")
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(branchCmd)
}
