package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/task"
	"github.com/cenkalti/work/internal/paths"
)

func RunGoal(ctx *location.Location, name string) error {
	branch := name
	wtPath := ctx.WorktreePath(name)
	created, err := git.CreateWorktree(ctx.RootRepo, wtPath, branch, git.DefaultBranch(ctx.RootRepo))
	if err != nil {
		return fmt.Errorf("creating goal worktree: %w", err)
	}

	success := false
	defer func() {
		if created && !success {
			_ = git.RemoveWorktree(ctx.RootRepo, wtPath)
		}
	}()

	spacePath := paths.GoalWorkspace(ctx.RootRepo, name)
	if err := os.MkdirAll(spacePath, 0755); err != nil {
		return fmt.Errorf("creating goal workspace: %w", err)
	}

	wsLink := filepath.Join(wtPath, "workspace")
	if _, err := os.Lstat(wsLink); os.IsNotExist(err) {
		if err := os.Symlink(spacePath, wsLink); err != nil {
			return fmt.Errorf("creating workspace symlink: %w", err)
		}
	}

	if err := os.WriteFile(filepath.Join(wtPath, "CLAUDE.md"), []byte(agent.GoalClaudeMD(name)), 0644); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}

	mcpJSON := `{
  "mcpServers": {
    "work": {
      "command": "work",
      "args": ["mcp"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(wtPath, ".mcp.json"), []byte(mcpJSON), 0644); err != nil {
		return fmt.Errorf("writing .mcp.json: %w", err)
	}

	success = true
	return ExecClaude(wtPath)
}

func RunTask(ctx *location.Location, goalBranch, taskID string) error {
	spacePath := paths.GoalWorkspace(ctx.RootRepo, goalBranch)

	taskData, err := os.ReadFile(filepath.Join(spacePath, "tasks", taskID+".json"))
	if err != nil {
		return fmt.Errorf("reading task file: %w", err)
	}
	var t task.Task
	if err := json.Unmarshal(taskData, &t); err != nil {
		return fmt.Errorf("parsing task file: %w", err)
	}

	if t.Status != task.StatusCompleted {
		t.Status = task.StatusActive
		if err := t.WriteToFile(filepath.Join(spacePath, "tasks")); err != nil {
			return fmt.Errorf("updating task status: %w", err)
		}
	}

	branch := fmt.Sprintf("%s.%s", goalBranch, t.ID)
	wtPath := ctx.WorktreePath(branch)
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

	tSpacePath := paths.TaskWorkspace(ctx.RootRepo, goalBranch, t.ID)
	if err := os.MkdirAll(tSpacePath, 0755); err != nil {
		return fmt.Errorf("creating task workspace: %w", err)
	}

	wsLink := filepath.Join(wtPath, "workspace")
	if _, err := os.Lstat(wsLink); os.IsNotExist(err) {
		if err := os.Symlink(tSpacePath, wsLink); err != nil {
			return fmt.Errorf("creating workspace symlink: %w", err)
		}
	}

	goalPath := filepath.Join(spacePath, "goal.md")
	goal, err := os.ReadFile(goalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read %s: %v\n", goalPath, err)
	}
	claudeMD := agent.TaskClaudeMD(goalBranch, string(goal), &t)
	if err := os.WriteFile(filepath.Join(wtPath, "CLAUDE.md"), []byte(claudeMD), 0644); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}

	success = true
	return ExecClaude(wtPath)
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
