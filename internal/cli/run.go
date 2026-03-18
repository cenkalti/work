package cli

import (
	"os"

	"github.com/cenkalti/work/internal/session"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [name]",
		Short: "Start a Claude Code session",
		Long: `work run                # start session in current worktree
work run <task>         # create root task worktree and start session
work run <task.subtask> # create child task worktree and start session`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			if len(args) == 0 {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				return session.ExecClaude(cwd)
			}
			return session.Run(loc, args[0])
		},
	}
}
