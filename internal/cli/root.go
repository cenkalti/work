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

	worktreeGroup := &cobra.Group{ID: "worktree", Title: "Worktree Commands:"}
	taskGroup := &cobra.Group{ID: "task", Title: "Task Commands:"}
	cmd.AddGroup(worktreeGroup, taskGroup)

	for _, c := range []*cobra.Command{
		runCmd(),
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
		setStatusCmd(),
	} {
		c.GroupID = "task"
		cmd.AddCommand(c)
	}

	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}
