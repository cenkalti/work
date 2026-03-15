package cli

import "path/filepath"

// WorktreeRootPath returns the path to the root directory containing all worktrees.
func WorktreeRootPath(root string) string {
	return filepath.Join(root, ".work", "tree")
}

// GoalWorktreePath returns the path to a goal's worktree directory.
func GoalWorktreePath(root, goal string) string {
	return filepath.Join(root, ".work", "tree", goal)
}

// TaskWorktreePath returns the path to a task's worktree directory.
func TaskWorktreePath(root, goal, taskID string) string {
	return filepath.Join(root, ".work", "tree", goal+"."+taskID)
}

// GoalSpacePath returns the path to a goal's workspace directory.
func GoalSpacePath(root, goal string) string {
	return filepath.Join(root, ".work", "space", goal)
}

// TaskSpacePath returns the path to a task's workspace directory.
func TaskSpacePath(root, goal, taskID string) string {
	return filepath.Join(root, ".work", "space", goal+"."+taskID)
}
