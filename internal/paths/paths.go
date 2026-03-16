package paths

import (
	"path/filepath"
	"strings"
)

func WorktreeRoot(root string) string {
	return filepath.Join(root, ".work", "tree")
}

func Worktree(root, branch string) string {
	return filepath.Join(root, ".work", "tree", branch)
}

func Workspace(root, branch string) string {
	return filepath.Join(root, ".work", "space", branch)
}

func TasksDir(root, branch string) string {
	return filepath.Join(Workspace(root, branch), "tasks")
}

// ParentBranch returns everything before the last dot. Returns "" for root tasks.
func ParentBranch(branch string) string {
	if i := strings.LastIndex(branch, "."); i >= 0 {
		return branch[:i]
	}
	return ""
}

// BranchID returns the last component after the last dot. Returns the full branch for root tasks.
func BranchID(branch string) string {
	if i := strings.LastIndex(branch, "."); i >= 0 {
		return branch[i+1:]
	}
	return branch
}
