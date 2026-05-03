package session

import (
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/task"
)

// Create sets up a worktree and workspace. wt.Name is the dot-separated task
// identity used for paths; `branch` is the git ref (may carry a prefix).
// If noBranch is true, no new git branch is created; the worktree checks out
// the repo's default branch directly (force, so it can coexist with the root
// repo's checkout). For child tasks, the task file is marked active.
// Returns the worktree path.
func Create(wt domain.Worktree, branch string, noBranch bool) (string, error) {
	parentName := domain.ParentBranchName(wt.Name)
	taskID := domain.BranchID(wt.Name)
	repo := wt.Repo()

	if parentName != "" {
		parent := domain.Worktree{RepoPath: wt.RepoPath, Name: parentName}
		if err := setTaskActive(parent.TasksDir(), taskID); err != nil {
			return "", err
		}
	}

	wtPath := wt.Path()
	var created bool
	if noBranch {
		if _, statErr := os.Stat(wtPath); statErr != nil {
			created = true
		}
		if err := git.CreateWorktreeOnBranch(repo.Path, wtPath, git.DefaultBranch(repo.Path)); err != nil {
			return "", fmt.Errorf("creating worktree: %w", err)
		}
	} else {
		var err error
		created, err = git.CreateWorktree(repo.Path, wtPath, branch, git.DefaultBranch(repo.Path))
		if err != nil {
			return "", fmt.Errorf("creating worktree: %w", err)
		}
	}

	success := false
	defer func() {
		if created && !success {
			_ = git.RemoveWorktree(repo.Path, wtPath)
		}
	}()

	if _, err := repo.EnsureProject(); err != nil {
		return "", err
	}
	spacePath := wt.WorkspacePath()
	if err := os.MkdirAll(spacePath, 0755); err != nil {
		return "", fmt.Errorf("creating workspace: %w", err)
	}

	wsLink := wt.WorkspaceLink()
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
