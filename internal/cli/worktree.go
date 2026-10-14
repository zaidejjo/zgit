package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

var (
	worktreeAddBranch string
	worktreeAddForce  bool
	worktreeAddDetach bool
	worktreeRmForce   bool
	worktreeLockMsg   string
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage linked working trees",
	Long: `List, add, remove, lock, and prune git worktrees.

A worktree lets you check out multiple branches at once in
separate directories without stashing or switching.`,
}

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		list, err := gitExec.WorktreeList(ctx)
		if err != nil {
			return fmt.Errorf("list worktrees: %w", err)
		}
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No worktrees found")
			return nil
		}
		for _, w := range list {
			fmt.Fprintln(os.Stdout, formatWorktree(w))
		}
		return nil
	},
}

var worktreeAddCmd = &cobra.Command{
	Use:   "add <path> [<ref>]",
	Short: "Add a new worktree",
	Long: `Add a worktree at <path> checking out <ref> (branch, tag, or commit).

Use --new-branch to create a new branch for the worktree:
  zgit worktree add ../hotfix --new-branch hotfix/login

Use --detach to check out HEAD detached:
  zgit worktree add ../detached abc123 --detach`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		path := args[0]
		ref := ""
		if len(args) > 1 {
			ref = args[1]
		}
		// --new-branch <name> creates a new branch for the worktree.
		createBranch := worktreeAddBranch != ""
		if createBranch {
			ref = worktreeAddBranch
		}
		opts := models.WorktreeAddOptions{
			CreateBranch: createBranch,
			Force:        worktreeAddForce,
			Detach:       worktreeAddDetach,
		}
		if err := gitExec.WorktreeAdd(ctx, path, ref, opts); err != nil {
			return fmt.Errorf("add worktree: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Added worktree at %s\n", path)
		return nil
	},
}

var worktreeRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove a worktree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := gitExec.WorktreeRemove(ctx, args[0], worktreeRmForce); err != nil {
			return fmt.Errorf("remove worktree: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Removed worktree at %s\n", args[0])
		return nil
	},
}

var worktreeLockCmd = &cobra.Command{
	Use:   "lock <path>",
	Short: "Lock a worktree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := gitExec.WorktreeLock(ctx, args[0], worktreeLockMsg); err != nil {
			return fmt.Errorf("lock worktree: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Locked worktree at %s\n", args[0])
		return nil
	},
}

var worktreeUnlockCmd = &cobra.Command{
	Use:   "unlock <path>",
	Short: "Unlock a worktree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := gitExec.WorktreeUnlock(ctx, args[0]); err != nil {
			return fmt.Errorf("unlock worktree: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Unlocked worktree at %s\n", args[0])
		return nil
	},
}

var worktreePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune worktree metadata for deleted directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := gitExec.WorktreePrune(ctx); err != nil {
			return fmt.Errorf("prune worktrees: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Pruned worktrees")
		return nil
	},
}

var worktreeRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair worktree metadata (git >= 2.27)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := gitExec.WorktreeRepair(ctx); err != nil {
			return fmt.Errorf("repair worktrees: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Repaired worktrees")
		return nil
	},
}

func formatWorktree(w *models.Worktree) string {
	label := w.Path
	switch {
	case w.IsMain:
		label += "  [main]"
	case w.IsBare:
		label += "  [bare]"
	}
	if w.Branch != "" {
		label += "  (" + w.Branch + ")"
	} else if w.IsDetached {
		label += "  (detached)"
	}
	if len(w.Head) >= 8 {
		label += "  " + w.Head[:8]
	} else if w.Head != "" {
		label += "  " + w.Head
	}
	if w.Locked {
		if w.LockReason != "" {
			label += "  [locked: " + w.LockReason + "]"
		} else {
			label += "  [locked]"
		}
	}
	if w.Prunable {
		label += "  [prunable]"
	}
	return label
}

func init() {
	worktreeAddCmd.Flags().StringVar(&worktreeAddBranch, "new-branch", "", "create a new branch for the worktree (value = branch name)")
	worktreeAddCmd.Flags().BoolVar(&worktreeAddForce, "force", false, "allow checking out an already-checked-out branch")
	worktreeAddCmd.Flags().BoolVar(&worktreeAddDetach, "detach", false, "check out HEAD detached")
	worktreeRemoveCmd.Flags().BoolVarP(&worktreeRmForce, "force", "f", false, "force removal even with dirty or locked state")
	worktreeLockCmd.Flags().StringVarP(&worktreeLockMsg, "message", "m", "", "optional reason for locking")

	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeAddCmd)
	worktreeCmd.AddCommand(worktreeRemoveCmd)
	worktreeCmd.AddCommand(worktreeLockCmd)
	worktreeCmd.AddCommand(worktreeUnlockCmd)
	worktreeCmd.AddCommand(worktreePruneCmd)
	worktreeCmd.AddCommand(worktreeRepairCmd)
}
