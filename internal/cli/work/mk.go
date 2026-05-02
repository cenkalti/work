package work

import (
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/session"
	"github.com/spf13/cobra"
)

func mkCmd() *cobra.Command {
	var noBranch bool
	cmd := &cobra.Command{
		Use:   "mk <name>",
		Short: "Create a worktree and workspace",
		Long: `work mk myfeature             # create root task worktree on a new branch
work mk myfeature.subtask     # create child task worktree on a new branch
work mk --no-branch myfeature # create worktree on main/master without a new branch

Names are absolute (dot-separated branch paths).
If WORK_BRANCH_PREFIX is set, its value is prepended to the new branch name
(ignored with --no-branch).
Prints the worktree path on success.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			branch := os.Getenv("WORK_BRANCH_PREFIX") + name
			wtPath, err := session.Create(loc, name, branch, noBranch)
			if err != nil {
				return err
			}
			fmt.Println(wtPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&noBranch, "no-branch", false, "use the default branch without creating a new one")
	return cmd
}
