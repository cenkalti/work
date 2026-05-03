package work

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a worktree and its branch",
		Long: `work rm myfeature            # remove root task worktree and branch
work rm myfeature.subtask    # remove child task worktree and branch

Names are absolute (dot-separated branch paths).
The task workspace at ~/.work/space/<project>/<task>/ is preserved unless it is empty (in which case it is removed too).`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			repo := loc.Repo
			branch := args[0]
			if project, sub, ok := strings.Cut(branch, "/"); ok {
				projectPath := filepath.Join(domain.ProjectsDir(), project)
				if info, err := os.Stat(projectPath); err == nil && info.IsDir() {
					repo = domain.Repo{Path: projectPath}
					branch = sub
				}
			}
			taskID := domain.BranchID(branch)
			wt := domain.Worktree{RepoPath: repo.Path, Name: branch}
			wtPath := wt.Path()

			if !yes && isTerminal() {
				fmt.Printf("Remove task %s and its worktree? [y/N] ", taskID)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				if !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := git.RemoveWorktreeIfExists(repo.Path, wtPath); err != nil {
				return fmt.Errorf("remove worktree: %w", err)
			}
			if err := git.DeleteBranchIfExists(repo.Path, branch); err != nil {
				return fmt.Errorf("delete branch: %w", err)
			}
			if removed, err := removeEmptyWorkspace(wt); err != nil {
				fmt.Fprintf(os.Stderr, "warn: workspace cleanup: %v\n", err)
			} else if removed {
				fmt.Printf("Task %s removed (workspace was empty).\n", taskID)
				return nil
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

// removeEmptyWorkspace removes the task's workspace dir if it has no
// content beyond `.DS_Store` (macOS auto-clutter). Returns true if removed.
func removeEmptyWorkspace(wt domain.Worktree) (bool, error) {
	wsPath := wt.WorkspacePath()
	entries, err := os.ReadDir(wsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, e := range entries {
		if e.Name() == ".DS_Store" {
			continue
		}
		return false, nil
	}
	if err := os.RemoveAll(wsPath); err != nil {
		return false, err
	}
	return true, nil
}
