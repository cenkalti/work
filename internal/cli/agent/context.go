package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
)

// isWorkManaged reports whether the root repo has a .work/ directory.
func isWorkManaged(rootRepo string) bool {
	_, err := os.Stat(filepath.Join(rootRepo, ".work"))
	return err == nil
}

func printTaskContext(rootRepo, branch string) error {
	parentBranch := paths.ParentBranch(branch)
	taskID := paths.BranchID(branch)

	if parentBranch == "" {
		printRootTaskContext(branch)
		return nil
	}

	tasksDir := paths.TasksDir(rootRepo, parentBranch)
	t, err := task.Load(tasksDir, taskID)
	if err != nil {
		return err
	}

	parentContext := readFileContents(filepath.Join(paths.Workspace(rootRepo, parentBranch), "plan.md"))
	printChildTaskContext(parentContext, t)
	return nil
}

func printRootTaskContext(branch string) {
	fmt.Printf("# Task: %s\n\n", branch)
	fmt.Printf("You are working on task **%s**. Work is a multi-task orchestrator that decomposes plans into subtasks with dependencies, then runs each subtask as a separate Claude Code instance in its own git worktree.\n\n", branch)
	fmt.Printf("Use the `/work:plan` slash command to start the planning workflow.\n\n")
	fmt.Printf("## Workspace\n\n")
	fmt.Printf("Your workspace is at `workspace/`. Use it for all planning documents.\n\n")
	printAvailableCommands()
	fmt.Printf("## Key Files\n\n")
	fmt.Println("- `workspace/plan.md` — The implementation plan (created during `/work:plan`)")
	fmt.Println("- `workspace/tasks/` — Subtask JSON files (created via the task MCP tool)")
}

func printChildTaskContext(parentContext string, t *task.Task) {
	if parentContext != "" {
		fmt.Printf("# Parent Task\n\n")
		fmt.Println(strings.TrimSpace(parentContext))
		fmt.Println()
	}

	fmt.Printf("# Your Task\n\n")
	fmt.Printf("**ID:** %s\n\n", t.ID)
	fmt.Printf("**Summary:** %s\n", t.Summary)

	if t.Description != "" {
		fmt.Printf("\n**Description:** %s\n", t.Description)
	}
	if len(t.Files) > 0 {
		fmt.Printf("\n**Files:** %s\n", strings.Join(t.Files, ", "))
	}
	if len(t.Acceptance) > 0 {
		fmt.Printf("\n**Acceptance Criteria:**\n")
		for _, a := range t.Acceptance {
			fmt.Printf("- %s\n", a)
		}
	}
	if t.Context != "" {
		fmt.Printf("\n**Context:** %s\n", t.Context)
	}

	fmt.Printf("\n## Workspace\n\n")
	fmt.Println("Your workspace is at `workspace/`.")
	fmt.Printf("Use it for scratch files, intermediate outputs, and notes.\n\n")
	fmt.Printf("**Work Log:** `workspace/log.md`\n\n")
	printAvailableCommands()
}

func printAvailableCommands() {
	fmt.Printf("## Available Commands\n\n")
	fmt.Println("```bash")
	fmt.Println("task ls                        # List subtasks")
	fmt.Println("task ls --ready                # Subtasks ready to work on")
	fmt.Println("task show <id>                   # Show details of a subtask")
	fmt.Println("task tree [id]                   # Show subtask dependency tree")
	fmt.Println("task set-status <id> <status>    # Set subtask status")
	fmt.Println("work mk <id>                     # Create a worktree for a subtask")
	fmt.Println("agent run                        # Start a Claude Code session")
	fmt.Println("work rm <id>                     # Remove a subtask worktree")
	fmt.Printf("```\n\n")
}

func readFileContents(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}
