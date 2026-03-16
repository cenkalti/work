package cli

import (
	"github.com/cenkalti/work/internal/session"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "run <name>",
		Short:             "Start a Claude Code session for a goal or task",
		Long:              "If name contains a dot (goal.task) or you're in a goal worktree, runs a task.\nOtherwise runs a goal.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			goal, taskID := loc.ResolveName(args[0])
			if taskID != "" {
				return session.RunTask(loc, goal, taskID)
			}
			return session.RunGoal(loc, goal)
		},
	}
}
