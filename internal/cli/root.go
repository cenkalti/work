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

	sessionGroup := &cobra.Group{ID: "session", Title: "Session Commands:"}
	worktreeGroup := &cobra.Group{ID: "worktree", Title: "Worktree Commands:"}
	taskGroup := &cobra.Group{ID: "task", Title: "Task Commands:"}
	cmd.AddGroup(sessionGroup, worktreeGroup, taskGroup)

	for _, c := range []*cobra.Command{
		runCmd(),
		mergeCmd(),
	} {
		c.GroupID = "session"
		cmd.AddCommand(c)
	}

	for _, c := range []*cobra.Command{
		nameCmd(),
		lsCmd(),
		mvCmd(),
		removeCmd(),
		cdCmd(),
		mcpCmd(),
		contextCmd(),
	} {
		c.GroupID = "worktree"
		cmd.AddCommand(c)
	}

	for _, c := range []*cobra.Command{
		listCmd(),
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
