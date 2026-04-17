package work

import (
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "Multi-task orchestration for Claude Code",
	}

	cmd.AddCommand(
		mkCmd(),
		idCmd(),
		lsCmd(),
		mvCmd(),
		removeCmd(),
		cdCmd(),
	)

	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
