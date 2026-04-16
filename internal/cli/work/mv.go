package work

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/paths"
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
			loc := detectLocation(cmd)
			if !loc.IsRoot() {
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

			root := loc.RootRepo

			// Preflight checks.
			if err := validateMove(root, src, dst); err != nil {
				return err
			}

			// Move from root (src == "")
			if src == "" {
				return moveFromRoot(root, dst)
			}

			// Move to root (dst == "")
			if dst == "" {
				return moveToRoot(root, src)
			}

			// Collect all branches that need renaming (src + children).
			branches, err := collectBranches(root, src)
			if err != nil {
				return err
			}

			// General move: rename branches, move worktrees, move workspaces.
			for _, oldBranch := range branches {
				newBranch := dst + strings.TrimPrefix(oldBranch, src)
				if err := moveTask(root, oldBranch, newBranch); err != nil {
					return err
				}
			}

			// Move task JSON from old parent to new parent.
			if err := moveTaskJSON(root, src, dst); err != nil {
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
	loc := detectLocation(cmd)
	worktrees, err := git.ListWorktrees(loc.RootRepo)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	wtRoot, err := filepath.EvalSymlinks(loc.WorktreeRoot())
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
func validateMove(root, src, dst string) error {
	if src == "" {
		// Moving from root: workspace must exist.
		wsPath := paths.WorkspaceLink(root)
		if _, err := os.Lstat(wsPath); err != nil {
			return fmt.Errorf("no workspace found at root")
		}
		// Destination workspace must not exist.
		dstSpace := paths.Workspace(root, dst)
		if _, err := os.Stat(dstSpace); err == nil {
			return fmt.Errorf("destination workspace already exists: %s", dst)
		}
		// Destination worktree must not exist.
		dstWT := paths.Worktree(root, dst)
		if _, err := os.Stat(dstWT); err == nil {
			return fmt.Errorf("destination worktree already exists: %s", dst)
		}
		return nil
	}

	if dst == "" {
		// Moving to root: source workspace must exist.
		srcSpace := paths.Workspace(root, src)
		if _, err := os.Stat(srcSpace); err != nil {
			return fmt.Errorf("workspace not found for %s", src)
		}
		// Root workspace must not exist.
		wsPath := paths.WorkspaceLink(root)
		if _, err := os.Lstat(wsPath); err == nil {
			return fmt.Errorf("workspace already exists at root; remove it first")
		}
		return nil
	}

	// General move: source workspace or worktree must exist.
	srcSpace := paths.Workspace(root, src)
	srcWT := paths.Worktree(root, src)
	srcSpaceExists := false
	srcWTExists := false
	if _, err := os.Stat(srcSpace); err == nil {
		srcSpaceExists = true
	}
	if _, err := os.Stat(srcWT); err == nil {
		srcWTExists = true
	}
	if !srcSpaceExists && !srcWTExists {
		return fmt.Errorf("source task not found: %s (no workspace or worktree)", src)
	}

	// Destination workspace must not exist.
	dstSpace := paths.Workspace(root, dst)
	if _, err := os.Stat(dstSpace); err == nil {
		return fmt.Errorf("destination workspace already exists: %s", dst)
	}
	// Destination worktree must not exist.
	dstWT := paths.Worktree(root, dst)
	if _, err := os.Stat(dstWT); err == nil {
		return fmt.Errorf("destination worktree already exists: %s", dst)
	}

	return nil
}

// collectBranches returns the src branch and all child branches (src.*).
// For src == "", returns nil (root has no branch).
func collectBranches(root, src string) ([]string, error) {
	if src == "" {
		return nil, nil
	}
	worktrees, err := git.ListWorktrees(root)
	if err != nil {
		return nil, err
	}
	wtRoot := paths.WorktreeRoot(root)
	prefix := wtRoot
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
func moveTask(root, oldBranch, newBranch string) error {
	// Rename git branch.
	if err := git.RenameBranch(root, oldBranch, newBranch); err != nil {
		return fmt.Errorf("renaming branch %s → %s: %w", oldBranch, newBranch, err)
	}

	// Move worktree on disk.
	oldWT := paths.Worktree(root, oldBranch)
	newWT := paths.Worktree(root, newBranch)
	if _, err := os.Stat(oldWT); err == nil {
		if err := git.MoveWorktree(root, oldWT, newWT); err != nil {
			return fmt.Errorf("moving worktree: %w", err)
		}
		// Update workspace symlink in the worktree.
		newSpace := paths.Workspace(root, newBranch)
		wsLink := paths.WorkspaceLink(newWT)
		_ = os.Remove(wsLink)
		if err := os.Symlink(newSpace, wsLink); err != nil {
			return fmt.Errorf("updating workspace symlink: %w", err)
		}
	}

	// Move workspace on disk.
	oldSpace := paths.Workspace(root, oldBranch)
	newSpace := paths.Workspace(root, newBranch)
	if _, err := os.Stat(oldSpace); err == nil {
		if err := os.Rename(oldSpace, newSpace); err != nil {
			return fmt.Errorf("moving workspace %s → %s: %w", oldSpace, newSpace, err)
		}
	}

	return nil
}

// moveTaskJSON moves the task JSON file from the old parent's tasks dir to the new parent's tasks dir.
func moveTaskJSON(root, oldBranch, newBranch string) error {
	oldParent := paths.ParentBranch(oldBranch)
	newParent := paths.ParentBranch(newBranch)
	oldID := paths.BranchID(oldBranch)
	newID := paths.BranchID(newBranch)

	if oldParent == "" && newParent == "" {
		return nil // root -> root rename, no task JSON
	}

	// Source task file.
	if oldParent != "" {
		oldFile := filepath.Join(paths.TasksDir(root, oldParent), oldID+".yaml")
		if _, err := os.Stat(oldFile); err != nil {
			return nil // no task file to move
		}
		if newParent != "" {
			newTasksDir := paths.TasksDir(root, newParent)
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
func moveFromRoot(root, dst string) error {
	wsPath := paths.WorkspaceLink(root)
	info, err := os.Lstat(wsPath)
	if err != nil {
		return fmt.Errorf("no workspace found at root: %w", err)
	}

	dstSpace := paths.Workspace(root, dst)
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
			target = filepath.Join(root, target)
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
	wtPath := paths.Worktree(root, dst)
	if _, err := git.CreateWorktree(root, wtPath, dst, git.DefaultBranch(root)); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	// Create workspace symlink in the new worktree.
	wsLink := paths.WorkspaceLink(wtPath)
	_ = os.Remove(wsLink)
	if err := os.Symlink(dstSpace, wsLink); err != nil {
		return fmt.Errorf("creating workspace symlink: %w", err)
	}

	fmt.Printf("Moved root workspace → %s\n", dst)
	return nil
}

// moveToRoot moves a named task's workspace to the root workspace (./workspace).
func moveToRoot(root, src string) error {
	srcSpace := paths.Workspace(root, src)
	wsPath := paths.WorkspaceLink(root)

	// Move workspace to root.
	if err := os.Rename(srcSpace, wsPath); err != nil {
		return err
	}

	// Clean up worktree and branch if they exist.
	wtPath := paths.Worktree(root, src)
	_ = git.RemoveWorktreeIfExists(root, wtPath)
	_ = git.DeleteBranchIfExists(root, src)

	fmt.Printf("Moved %s → root workspace\n", src)
	return nil
}
