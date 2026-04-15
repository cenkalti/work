package agent

import (
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent lifecycle and listing",
	}

	cmd.AddCommand(
		runCmd(),
		startCmd(),
		endCmd(),
		statusCmd(),
		lsCmd(),
	)

	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
