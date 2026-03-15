package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <name>",
		Short: "Start a Claude Code session for a goal or task",
		Long:  "If name contains a dot (goal.task) or you're in a goal worktree, runs a task.\nOtherwise runs a goal.",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := workContext(cmd)
			goal, taskID, isTask := ctx.ResolveName(args[0])
			if isTask {
				return runTask(ctx, goal, taskID)
			}
			return runGoal(ctx, goal)
		},
	}
}

func runGoal(ctx *WorkContext, name string) error {
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

	spacePath := goalSpacePath(ctx.RootRepo, name)
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

	success = true
	return execClaude(wtPath)
}

func runTask(ctx *WorkContext, goalBranch, taskID string) error {
	spacePath := goalSpacePath(ctx.RootRepo, goalBranch)

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

	tSpacePath := taskSpacePath(ctx.RootRepo, goalBranch, t.ID)
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
	return execClaude(wtPath)
}

func execClaude(wtPath string) error {
	if err := os.Chdir(wtPath); err != nil {
		return fmt.Errorf("changing to worktree dir: %w", err)
	}
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}
	return syscall.Exec(claudeBin, []string{"claude"}, os.Environ())
}
