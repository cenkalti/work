package cli

import (
	"os"

	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/session"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "run [name]",
		Short:             "Start a Claude Code session for a goal or task",
		Long:              "If name contains a dot (goal.task) or you're in a goal worktree, runs a task.\nOtherwise runs a goal.\nWith no argument from a goal or task worktree, re-attaches Claude Code to the current worktree.",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			if len(args) == 0 {
				if loc.Type == location.Goal || loc.Type == location.Task {
					cwd, err := os.Getwd()
					if err != nil {
						return err
					}
					return session.ExecClaude(cwd)
				}
				return cmd.Usage()
			}
			goal, taskID := loc.ResolveName(args[0])
			if taskID != "" {
				return session.RunTask(loc, goal, taskID)
			}
			return session.RunGoal(loc, goal)
		},
	}
}
