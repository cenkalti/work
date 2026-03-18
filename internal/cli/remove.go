package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a worktree and its branch",
		Long: `work rm myfeature            # remove root task worktree and branch
work rm myfeature.subtask    # remove child task worktree and branch

Names are absolute (dot-separated branch paths).`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			branch := args[0]
			taskID := paths.BranchID(branch)
			wtPath := loc.WorktreePath(branch)

			if !yes && isTerminal() {
				fmt.Printf("Remove task %s and its worktree? [y/N] ", taskID)
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
			fmt.Printf("Task %s removed.\n", taskID)
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
