package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
)

// Run sets up and launches a Claude Code session for the given branch.
// For root tasks (no dots in branch), a planning-focused session is started.
// For child tasks, the task file is read from the parent's tasks dir.
func Run(ctx *location.Location, branch string) error {
	parentBranch := paths.ParentBranch(branch)
	taskID := paths.BranchID(branch)

	var t *task.Task
	if parentBranch != "" {
		tasksDir := paths.TasksDir(ctx.RootRepo, parentBranch)
		taskData, err := os.ReadFile(filepath.Join(tasksDir, taskID+".json"))
		if err != nil {
			return fmt.Errorf("reading task file: %w", err)
		}
		t = &task.Task{}
		if err := json.Unmarshal(taskData, t); err != nil {
			return fmt.Errorf("parsing task file: %w", err)
		}
		if t.Status != task.StatusCompleted {
			t.Status = task.StatusActive
			if err := t.WriteToFile(tasksDir); err != nil {
				return fmt.Errorf("updating task status: %w", err)
			}
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

	wsLink := filepath.Join(wtPath, "workspace")
	if _, err := os.Lstat(wsLink); os.IsNotExist(err) {
		if err := os.Symlink(spacePath, wsLink); err != nil {
			return fmt.Errorf("creating workspace symlink: %w", err)
		}
	}

	var claudeMD string
	if t == nil {
		claudeMD = agent.RootTaskClaudeMD(branch)
	} else {
		parentContext := readParentContext(ctx.RootRepo, parentBranch)
		claudeMD = agent.TaskClaudeMD(branch, parentContext, t)
	}

	if err := os.WriteFile(filepath.Join(wtPath, "CLAUDE.md"), []byte(claudeMD), 0644); err != nil {
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

func readParentContext(rootRepo, parentBranch string) string {
	planPath := filepath.Join(paths.Workspace(rootRepo, parentBranch), "plan.md")
	content, err := os.ReadFile(planPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
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
