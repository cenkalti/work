package task

import (
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task management for work orchestrator",
	}

	cmd.AddCommand(
		listCmd(),
		showCmd(),
		editCmd(),
		treeCmd(),
		setStatusCmd(),
		rmCmd(),
		migrateCmd(),
		mcpCmd(),
	)

	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
