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
		psCmd(),
		inboxCmd(),
		jumpCmd(),
		mvCmd(),
		rmCmd(),
		dashCmd(),
		dashFocusCmd(),
	)
	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
