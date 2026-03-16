package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func mergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge",
		Short: "Merge the current task branch into its parent",
		Long: `work merge      # merge current task into parent task (or default branch for root tasks)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			if loc.IsRoot() {
				return fmt.Errorf("not in a task worktree")
			}

			parentBranch := paths.ParentBranch(loc.Branch)
			if parentBranch == "" {
				parentBranch = git.DefaultBranch(loc.RootRepo)
			}

			if err := checkClean(loc.RootRepo, loc.Branch); err != nil {
				return err
			}

			targetDir := mergeTargetDir(loc.RootRepo, parentBranch)
			return runMerge(targetDir, loc.Branch)
		},
	}
}

// mergeTargetDir returns the directory from which to run the merge.
// Uses the parent's worktree if it exists, otherwise the root repo.
func mergeTargetDir(rootRepo, parentBranch string) string {
	wtPath := paths.Worktree(rootRepo, parentBranch)
	if _, err := os.Stat(wtPath); err == nil {
		return wtPath
	}
	return rootRepo
}

func checkClean(rootRepo, branch string) error {
	wtPath := paths.Worktree(rootRepo, branch)
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wtPath
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("checking git status: %w", err)
	}
	if len(out) > 0 {
		return fmt.Errorf("uncommitted changes in worktree; commit or discard before merging")
	}
	return nil
}

func runMerge(targetDir, branch string) error {
	cmd := exec.Command("git", "merge", branch)
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("merge failed")
	}
	return nil
}
