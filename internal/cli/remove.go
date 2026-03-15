package cli

import (
	"fmt"

	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a goal or task worktree and branch",
		Long:  "If name contains a dot (goal.task) or you're in a goal worktree, removes a task.\nOtherwise removes a goal.",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := workContext(cmd)
			goal, taskID, isTask := ctx.ResolveName(args[0])

			if isTask {
				branch := fmt.Sprintf("%s.%s", goal, taskID)
				wtPath := ctx.WorktreePath(branch)
				if err := git.RemoveWorktreeIfExists(ctx.RootRepo, wtPath); err != nil {
					return fmt.Errorf("remove worktree: %w", err)
				}
				if err := git.DeleteBranchIfExists(ctx.RootRepo, branch); err != nil {
					return fmt.Errorf("delete branch: %w", err)
				}
				fmt.Printf("Task %s removed.\n", taskID)
			} else {
				wtPath := ctx.WorktreePath(goal)
				if err := git.RemoveWorktreeIfExists(ctx.RootRepo, wtPath); err != nil {
					return fmt.Errorf("remove worktree: %w", err)
				}
				if err := git.DeleteBranchIfExists(ctx.RootRepo, goal); err != nil {
					return fmt.Errorf("delete branch: %w", err)
				}
				fmt.Printf("Goal %s removed.\n", goal)
			}
			return nil
		},
	}
}
