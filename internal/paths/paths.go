package paths

import "path/filepath"

// WorktreeRoot returns the path to the root directory containing all worktrees.
func WorktreeRoot(root string) string {
	return filepath.Join(root, ".work", "tree")
}

// GoalWorktree returns the path to a goal's worktree directory.
func GoalWorktree(root, goal string) string {
	return filepath.Join(root, ".work", "tree", goal)
}

// TaskWorktree returns the path to a task's worktree directory.
func TaskWorktree(root, goal, taskID string) string {
	return filepath.Join(root, ".work", "tree", goal+"."+taskID)
}

// GoalWorkspace returns the path to a goal's workspace directory.
func GoalWorkspace(root, goal string) string {
	return filepath.Join(root, ".work", "space", goal)
}

// TaskWorkspace returns the path to a task's workspace directory.
func TaskWorkspace(root, goal, taskID string) string {
	return filepath.Join(root, ".work", "space", goal+"."+taskID)
}

// TasksDir returns the tasks directory for a goal.
func TasksDir(root, goal string) string {
	return filepath.Join(GoalWorkspace(root, goal), "tasks")
}
