package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a worktree and its branch",
		Long: `work rm <goal>          # remove goal worktree and branch
work rm <task>          # remove task worktree and branch (from goal worktree)
work rm <goal.task>     # remove task worktree and branch`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			goal, taskID := loc.ResolveName(args[0])

			var subject, branch, wtPath string
			if taskID != "" {
				branch = fmt.Sprintf("%s.%s", goal, taskID)
				wtPath = loc.WorktreePath(branch)
				subject = fmt.Sprintf("task %s", taskID)
			} else {
				branch = goal
				wtPath = loc.WorktreePath(goal)
				subject = fmt.Sprintf("goal %s", goal)
			}

			if !yes && isTerminal() {
				fmt.Printf("Remove %s and its worktree? [y/N] ", subject)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				if !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := git.RemoveWorktreeIfExists(loc.RootRepo, wtPath); err != nil {
				return fmt.Errorf("remove worktree: %w", err)
			}
			if err := git.DeleteBranchIfExists(loc.RootRepo, branch); err != nil {
				return fmt.Errorf("delete branch: %w", err)
			}
			fmt.Printf("%s removed.\n", strings.ToUpper(subject[:1])+subject[1:])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
