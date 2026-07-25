package cli

import (
	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	Long:  `Start the Bubble Tea-based terminal UI for an interactive Git & GitHub experience.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(gitExec)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
