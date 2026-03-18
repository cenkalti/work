package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func contextCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "context",
		Short:  "Print task context for the current worktree",
		Hidden: true,
		Long: `Prints task context to stdout for injection into Claude Code's conversation.

Install in ~/.claude/settings.json to automatically inject context at session start:

  {
    "hooks": {
      "SessionStart": [
        {
          "matcher": "",
          "hooks": [{"type": "command", "command": "work context"}]
        }
      ]
    }
  }

Exits silently if not inside a work-managed worktree.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			if loc.IsRoot() || !isWorkManaged(loc.RootRepo) {
				return nil
			}
			return printTaskContext(loc.RootRepo, loc.Branch)
		},
	}
}

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

	data, err := os.ReadFile(filepath.Join(paths.TasksDir(rootRepo, parentBranch), taskID+".json"))
	if err != nil {
		return fmt.Errorf("reading task file: %w", err)
	}
	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("parsing task: %w", err)
	}

	parentContext := readFileContents(filepath.Join(paths.Workspace(rootRepo, parentBranch), "plan.md"))
	printChildTaskContext(parentContext, &t)
	return nil
}

func printRootTaskContext(branch string) {
	fmt.Printf("# Task: %s\n\n", branch)
	fmt.Printf("You are working on task **%s**. Work is a multi-task orchestrator that decomposes plans into subtasks with dependencies, then runs each subtask as a separate Claude Code instance in its own git worktree.\n\n", branch)
	fmt.Printf("Use the `/work-plan` slash command to start the planning workflow.\n\n")
	fmt.Printf("## Workspace\n\n")
	fmt.Printf("Your workspace is at `workspace/`. Use it for all planning documents.\n\n")
	printAvailableCommands()
	fmt.Printf("## Key Files\n\n")
	fmt.Println("- `workspace/plan.md` — The implementation plan (created during `/work-plan`)")
	fmt.Println("- `workspace/tasks/` — Subtask JSON files (created via the work MCP tool)")
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
	fmt.Println("work ls                 # List subtasks with their status")
	fmt.Println("work tasks                       # List subtasks")
	fmt.Println("work tasks --ready               # Subtasks ready to work on")
	fmt.Println("work show <id>                   # Show details of a subtask")
	fmt.Println("work tree [id]                   # Show subtask dependency tree")
	fmt.Println("work set-status <id> <status>    # Set subtask status")
	fmt.Println("work run <id>                    # Start a Claude Code session for a subtask")
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
