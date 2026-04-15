package session

import (
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
)

// Create sets up a worktree and workspace for the given branch.
// For child tasks, the task file is marked active.
// Returns the worktree path.
func Create(ctx *location.Location, branch string) (string, error) {
	parentBranch := paths.ParentBranch(branch)
	taskID := paths.BranchID(branch)

	if parentBranch != "" {
		if err := setTaskActive(paths.TasksDir(ctx.RootRepo, parentBranch), taskID); err != nil {
			return "", err
		}
	}

	wtPath := paths.Worktree(ctx.RootRepo, branch)
	created, err := git.CreateWorktree(ctx.RootRepo, wtPath, branch, git.DefaultBranch(ctx.RootRepo))
	if err != nil {
		return "", fmt.Errorf("creating worktree: %w", err)
	}

	success := false
	defer func() {
		if created && !success {
			_ = git.RemoveWorktree(ctx.RootRepo, wtPath)
		}
	}()

	spacePath := paths.Workspace(ctx.RootRepo, branch)
	if err := os.MkdirAll(spacePath, 0755); err != nil {
		return "", fmt.Errorf("creating workspace: %w", err)
	}

	wsLink := paths.WorkspaceLink(wtPath)
	if _, err := os.Lstat(wsLink); os.IsNotExist(err) {
		if err := os.Symlink(spacePath, wsLink); err != nil {
			return "", fmt.Errorf("creating workspace symlink: %w", err)
		}
	}

	success = true
	return wtPath, nil
}

func setTaskActive(tasksDir, taskID string) error {
	t, err := task.Load(tasksDir, taskID)
	if err != nil {
		return err
	}
	if t.Status == task.StatusCompleted {
		return nil
	}
	t.Status = task.StatusActive
	if err := t.WriteToFile(tasksDir); err != nil {
		return fmt.Errorf("updating task status: %w", err)
	}
	return nil
}
