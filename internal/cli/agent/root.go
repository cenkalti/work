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
		setupCmd(),
		runCmd(),
		hookCmd(),
		lsCmd(),
		inboxCmd(),
	)
	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
