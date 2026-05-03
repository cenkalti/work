package work

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func mvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <src> <dst>",
		Short: "Move/rename a task to a new location in the hierarchy",
		Long: `work mv . foo              # promote root workspace to task "foo"
work mv foo bar            # rename root task
work mv foo.a foo.b        # rename subtask
work mv foo.a bar.a        # move subtask to different parent
work mv foo .              # move task back to root workspace

Must be run from the repo root. Names are absolute (dot-separated branch paths).
Use "." to refer to the root repo (no task).`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: mvCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			if loc.Worktree != nil {
				return fmt.Errorf("must be run from the repo root")
			}

			src := args[0]
			dst := args[1]

			// Validate: "." is allowed, otherwise must be a valid absolute name (no relative resolution).
			if src != "." && strings.Contains(src, "..") {
				return fmt.Errorf("invalid source name: %s", src)
			}
			if dst != "." && strings.Contains(dst, "..") {
				return fmt.Errorf("invalid destination name: %s", dst)
			}

			// Normalize "." to empty string internally.
			if src == "." {
				src = ""
			}
			if dst == "." {
				dst = ""
			}

			if src == dst {
				return fmt.Errorf("source and destination are the same")
			}

			repo := loc.Repo

			// Preflight checks.
			if err := validateMove(repo, src, dst); err != nil {
				return err
			}

			// Move from root (src == "")
			if src == "" {
				return moveFromRoot(repo, dst)
			}

			// Move to root (dst == "")
			if dst == "" {
				return moveToRoot(repo, src)
			}

			// Collect all branches that need renaming (src + children).
			branches, err := collectBranches(repo, src)
			if err != nil {
				return err
			}

			// General move: rename branches, move worktrees, move workspaces.
			for _, oldBranch := range branches {
				newBranch := dst + strings.TrimPrefix(oldBranch, src)
				if err := moveTask(repo, oldBranch, newBranch); err != nil {
					return err
				}
			}

			// Move task JSON from old parent to new parent.
			if err := moveTaskJSON(repo, src, dst); err != nil {
				return err
			}

			fmt.Printf("Moved %s → %s\n", src, dst)
			return nil
		},
	}
}

func mvCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	loc, err := detectLocation(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	worktrees, err := git.ListWorktrees(loc.Repo.Path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	wtRoot, err := filepath.EvalSymlinks(loc.Repo.WorktreeRoot())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	prefix := wtRoot
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	names := []string{"."}
	for _, path := range worktrees {
		if name, ok := strings.CutPrefix(path, prefix); ok {
			names = append(names, name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// validateMove checks preconditions before performing any move operation.
func validateMove(repo domain.Repo, src, dst string) error {
	dstWt := domain.Worktree{RepoPath: repo.Path, Name: dst}
	if src == "" {
		// Moving from root: workspace must exist.
		if _, err := os.Lstat(repo.WorkspaceLink()); err != nil {
			return fmt.Errorf("no workspace found at root")
		}
		// Destination workspace must not exist.
		if _, err := os.Stat(dstWt.WorkspacePath()); err == nil {
			return fmt.Errorf("destination workspace already exists: %s", dst)
		}
		// Destination worktree must not exist.
		if _, err := os.Stat(dstWt.Path()); err == nil {
			return fmt.Errorf("destination worktree already exists: %s", dst)
		}
		return nil
	}

	srcWt := domain.Worktree{RepoPath: repo.Path, Name: src}
	if dst == "" {
		// Moving to root: source workspace must exist.
		if _, err := os.Stat(srcWt.WorkspacePath()); err != nil {
			return fmt.Errorf("workspace not found for %s", src)
		}
		// Root workspace must not exist.
		if _, err := os.Lstat(repo.WorkspaceLink()); err == nil {
			return fmt.Errorf("workspace already exists at root; remove it first")
		}
		return nil
	}

	// General move: source workspace or worktree must exist.
	srcSpaceExists := false
	srcWTExists := false
	if _, err := os.Stat(srcWt.WorkspacePath()); err == nil {
		srcSpaceExists = true
	}
	if _, err := os.Stat(srcWt.Path()); err == nil {
		srcWTExists = true
	}
	if !srcSpaceExists && !srcWTExists {
		return fmt.Errorf("source task not found: %s (no workspace or worktree)", src)
	}

	// Destination workspace must not exist.
	if _, err := os.Stat(dstWt.WorkspacePath()); err == nil {
		return fmt.Errorf("destination workspace already exists: %s", dst)
	}
	// Destination worktree must not exist.
	if _, err := os.Stat(dstWt.Path()); err == nil {
		return fmt.Errorf("destination worktree already exists: %s", dst)
	}

	return nil
}

// collectBranches returns the src branch and all child branches (src.*).
// For src == "", returns nil (root has no branch).
func collectBranches(repo domain.Repo, src string) ([]string, error) {
	if src == "" {
		return nil, nil
	}
	worktrees, err := git.ListWorktrees(repo.Path)
	if err != nil {
		return nil, err
	}
	prefix := repo.WorktreeRoot()
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	var branches []string
	for _, wt := range worktrees {
		name, ok := strings.CutPrefix(wt, prefix)
		if !ok {
			continue
		}
		if name == src || strings.HasPrefix(name, src+".") {
			branches = append(branches, name)
		}
	}
	// Even if no worktree exists, include src itself for workspace/branch operations.
	if len(branches) == 0 {
		branches = []string{src}
	}
	return branches, nil
}

// moveTask renames a git branch, moves its worktree, and moves its workspace.
func moveTask(repo domain.Repo, oldBranch, newBranch string) error {
	// Skip the branch rename if the old branch equals the new branch with the
	// WORK_BRANCH_PREFIX prepended — the user is dropping the prefix from the
	// worktree path while leaving the underlying branch name alone.
	if prefix := os.Getenv("WORK_BRANCH_PREFIX"); prefix == "" || prefix+newBranch != oldBranch {
		if err := git.RenameBranch(repo.Path, oldBranch, newBranch); err != nil {
			return fmt.Errorf("renaming branch %s → %s: %w", oldBranch, newBranch, err)
		}
	}

	oldWt := domain.Worktree{RepoPath: repo.Path, Name: oldBranch}
	newWt := domain.Worktree{RepoPath: repo.Path, Name: newBranch}

	// Move worktree on disk.
	if _, err := os.Stat(oldWt.Path()); err == nil {
		if err := git.MoveWorktree(repo.Path, oldWt.Path(), newWt.Path()); err != nil {
			return fmt.Errorf("moving worktree: %w", err)
		}
		// Update workspace symlink in the worktree.
		wsLink := newWt.WorkspaceLink()
		if err := os.Remove(wsLink); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing old workspace symlink: %w", err)
		}
		if err := os.Symlink(newWt.WorkspacePath(), wsLink); err != nil {
			return fmt.Errorf("updating workspace symlink: %w", err)
		}
	}

	// Move workspace on disk.
	if _, err := os.Stat(oldWt.WorkspacePath()); err == nil {
		if err := os.Rename(oldWt.WorkspacePath(), newWt.WorkspacePath()); err != nil {
			return fmt.Errorf("moving workspace %s → %s: %w", oldWt.WorkspacePath(), newWt.WorkspacePath(), err)
		}
	}

	return nil
}

// moveTaskJSON moves the task JSON file from the old parent's tasks dir to the new parent's tasks dir.
func moveTaskJSON(repo domain.Repo, oldBranch, newBranch string) error {
	oldParent := domain.ParentBranchName(oldBranch)
	newParent := domain.ParentBranchName(newBranch)
	oldID := domain.BranchID(oldBranch)
	newID := domain.BranchID(newBranch)

	if oldParent == "" && newParent == "" {
		return nil // root -> root rename, no task JSON
	}

	// Source task file.
	if oldParent != "" {
		oldParentWt := domain.Worktree{RepoPath: repo.Path, Name: oldParent}
		oldFile := filepath.Join(oldParentWt.TasksDir(), oldID+".yaml")
		if _, err := os.Stat(oldFile); err != nil {
			return nil // no task file to move
		}
		if newParent != "" {
			newParentWt := domain.Worktree{RepoPath: repo.Path, Name: newParent}
			newTasksDir := newParentWt.TasksDir()
			if err := os.MkdirAll(newTasksDir, 0755); err != nil {
				return err
			}
			newFile := filepath.Join(newTasksDir, newID+".yaml")
			return os.Rename(oldFile, newFile)
		}
		// Moving to root — just remove from old parent (root tasks don't have JSON files).
		return os.Remove(oldFile)
	}

	return nil
}

// moveFromRoot moves the root workspace (./workspace) into a named task.
func moveFromRoot(repo domain.Repo, dst string) error {
	wsPath := repo.WorkspaceLink()
	info, err := os.Lstat(wsPath)
	if err != nil {
		return fmt.Errorf("no workspace found at root: %w", err)
	}

	if _, err := repo.EnsureProject(); err != nil {
		return err
	}
	dstWt := domain.Worktree{RepoPath: repo.Path, Name: dst}
	dstSpace := dstWt.WorkspacePath()
	if err := os.MkdirAll(filepath.Dir(dstSpace), 0755); err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		// It's a symlink — read target, remove link, rename target dir.
		target, err := os.Readlink(wsPath)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(repo.Path, target)
		}
		if err := os.Remove(wsPath); err != nil {
			return err
		}
		if err := os.Rename(target, dstSpace); err != nil {
			return err
		}
	} else if info.IsDir() {
		// It's a real directory — move it.
		if err := os.Rename(wsPath, dstSpace); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("workspace is not a directory or symlink")
	}

	// Create worktree and branch for the destination.
	if _, err := git.CreateWorktree(repo.Path, dstWt.Path(), dst, git.DefaultBranch(repo.Path)); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	// Create workspace symlink in the new worktree.
	wsLink := dstWt.WorkspaceLink()
	if err := os.Remove(wsLink); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing old workspace symlink: %w", err)
	}
	if err := os.Symlink(dstSpace, wsLink); err != nil {
		return fmt.Errorf("creating workspace symlink: %w", err)
	}

	fmt.Printf("Moved root workspace → %s\n", dst)
	return nil
}

// moveToRoot moves a named task's workspace to the root workspace (./workspace).
func moveToRoot(repo domain.Repo, src string) error {
	srcWt := domain.Worktree{RepoPath: repo.Path, Name: src}

	// Move workspace to root.
	if err := os.Rename(srcWt.WorkspacePath(), repo.WorkspaceLink()); err != nil {
		return err
	}

	// Clean up worktree and branch if they exist.
	if err := git.RemoveWorktreeIfExists(repo.Path, srcWt.Path()); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}
	if err := git.DeleteBranchIfExists(repo.Path, src); err != nil {
		return fmt.Errorf("deleting branch: %w", err)
	}

	fmt.Printf("Moved %s → root workspace\n", src)
	return nil
}
