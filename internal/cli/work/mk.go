package work

import (
	"fmt"

	"github.com/cenkalti/work/internal/session"
	"github.com/spf13/cobra"
)

func mkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mk <name>",
		Short: "Create a worktree and workspace",
		Long: `work mk myfeature          # create root task worktree
work mk myfeature.subtask  # create child task worktree

Names are absolute (dot-separated branch paths).
Prints the worktree path on success.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			wtPath, err := session.Create(loc, args[0])
			if err != nil {
				return err
			}
			fmt.Println(wtPath)
			return nil
		},
	}
}
