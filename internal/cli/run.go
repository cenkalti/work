package cli

import (
	"os"

	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/session"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [name]",
		Short: "Start a Claude Code session",
		Long: `work run                # start session in current worktree
work run <goal>         # create goal worktree and start session
work run <task>         # create task worktree and start session (from goal worktree)
work run <goal.task>    # create task worktree and start session`,
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
