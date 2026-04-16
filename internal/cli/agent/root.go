package agent

import (
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent lifecycle and listing",
	}

	hook := &cobra.Command{
		Use:    "hook",
		Short:  "Hook commands for Claude Code integration",
		Hidden: true,
	}
	hook.AddCommand(
		contextCmd(),
		startCmd(),
		endCmd(),
		statusCmd(),
		bashCheckCmd(),
		notifyCmd(),
	)

	cmd.AddCommand(
		setupCmd(),
		runCmd(),
		hook,
		lsCmd(),
		inboxCmd(),
	)

	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
