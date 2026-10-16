package cli

import (
	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/tui"
)

var tuiMouse bool

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	Long:  `Start the Bubble Tea-based terminal UI for an interactive Git & GitHub experience.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunWithMouse(gitExec, tuiMouse)
	},
}

func init() {
	tuiCmd.Flags().BoolVar(&tuiMouse, "mouse", false, "enable mouse wheel scrolling in lists")
	rootCmd.AddCommand(tuiCmd)
}
