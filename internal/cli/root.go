package cli

import (
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "work",
		Short:             "Multi-task orchestration for Claude Code",
		PersistentPreRunE: persistWorkContext,
	}

	generalGroup := &cobra.Group{ID: "general", Title: "General Commands:"}
	taskGroup := &cobra.Group{ID: "task", Title: "Task Commands:"}
	cmd.AddGroup(generalGroup, taskGroup)

	for _, c := range []*cobra.Command{
		runCmd(),
		listCmd(),
		removeCmd(),
		cdCmd(),
		mcpCmd(),
	} {
		c.GroupID = "general"
		cmd.AddCommand(c)
	}

	for _, c := range []*cobra.Command{
		decomposeCmd(),
		showCmd(),
		treeCmd(),
		readyCmd(),
		activeCmd(),
		completeCmd(),
	} {
		c.GroupID = "task"
		cmd.AddCommand(c)
	}

	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
