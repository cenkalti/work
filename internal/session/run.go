package session

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
)

// Run sets up and launches a Claude Code session for the given branch.
// For root tasks (no dots in branch), a planning-focused session is started.
// For child tasks, the task file is read from the parent's tasks dir and marked active.
func Run(ctx *location.Location, branch string) error {
	parentBranch := paths.ParentBranch(branch)
	taskID := paths.BranchID(branch)

	if parentBranch != "" {
		if err := setTaskActive(paths.TasksDir(ctx.RootRepo, parentBranch), taskID); err != nil {
			return err
		}
	}

	wtPath := paths.Worktree(ctx.RootRepo, branch)
	created, err := git.CreateWorktree(ctx.RootRepo, wtPath, branch, git.DefaultBranch(ctx.RootRepo))
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	success := false
	defer func() {
		if created && !success {
			_ = git.RemoveWorktree(ctx.RootRepo, wtPath)
		}
	}()

	spacePath := paths.Workspace(ctx.RootRepo, branch)
	if err := os.MkdirAll(spacePath, 0755); err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	wsLink := paths.WorkspaceLink(wtPath)
	if _, err := os.Lstat(wsLink); os.IsNotExist(err) {
		if err := os.Symlink(spacePath, wsLink); err != nil {
			return fmt.Errorf("creating workspace symlink: %w", err)
		}
	}

	success = true
	return ExecClaude(wtPath)
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

func ExecClaude(wtPath string) error {
	if err := os.Chdir(wtPath); err != nil {
		return fmt.Errorf("changing to worktree dir: %w", err)
	}
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}
	return syscall.Exec(claudeBin, []string{"claude"}, os.Environ())
}
