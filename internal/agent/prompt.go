package agent

import (
	"fmt"
	"strings"

	"github.com/cenkalti/work/internal/task"
)

// GoalClaudeMD returns the CLAUDE.md content written into goal worktrees so Claude knows about work commands.
func GoalClaudeMD(goalBranch string) string {
	return `# Work

You are working inside a goal worktree. Work is a multi-task orchestrator that decomposes plans into tasks with dependencies, then runs each task as a separate Claude Code instance in its own git worktree.

Use the ` + "`/work-plan`" + ` slash command to start the planning workflow.

## Workspace

Your workspace is at ` + "`.work/space/" + goalBranch + "/`" + `. Use it for all planning documents (goal.md, plan.md, research.md, etc.).

## Available Commands

` + "```" + `bash
# Decompose the plan into tasks (requires .work/space/<goal>/plan.md)
work decompose
work decompose -i "for milestone 1 only"

# Task management
work list               # List all tasks with their status
work tree               # Show task dependency tree
work tree <id>          # Show subtree rooted at a task
work ready              # Show tasks ready to work on (no pending dependencies)
work active             # Show tasks currently being worked on
work show <id>          # Show details of a specific task
work run <id>           # Start a Claude Code session for a task
work complete <id>      # Mark a task as complete
work remove <id>        # Remove a task
` + "```" + `

## Key Files

- ` + "`.work/space/" + goalBranch + "/goal.md`" + ` — The goal description
- ` + "`.work/space/" + goalBranch + "/plan.md`" + ` — The implementation plan (created during ` + "`/work-plan`" + `)
- ` + "`.work/space/" + goalBranch + "/tasks/`" + ` — Task JSON files (created by ` + "`work decompose`" + `)

## Cross-Goal Context

All goals share the ` + "`.work/space/`" + ` directory. You can read other goals' workspaces for context:
- ` + "`.work/space/<other-goal>/goal.md`" + `
- ` + "`.work/space/<other-goal>.<task-id>/log.md`" + `
`
}

func TaskClaudeMD(goalBranch, goal string, t *task.Task) string {
	var b strings.Builder

	if goal != "" {
		fmt.Fprintf(&b, "# Goal\n\n%s\n\n", strings.TrimSpace(goal))
	}

	fmt.Fprintf(&b, "# Your Task\n\n")
	fmt.Fprintf(&b, "**ID:** %s\n\n", t.ID)
	fmt.Fprintf(&b, "**Summary:** %s\n\n", t.TaskSummary)
	if t.Description != "" {
		fmt.Fprintf(&b, "**Description:** %s\n\n", t.Description)
	}
	if len(t.Files) > 0 {
		fmt.Fprintf(&b, "**Files:** %s\n\n", strings.Join(t.Files, ", "))
	}
	if len(t.Acceptance) > 0 {
		fmt.Fprintf(&b, "**Acceptance Criteria:**\n")
		for _, a := range t.Acceptance {
			fmt.Fprintf(&b, "- %s\n", a)
		}
		fmt.Fprintf(&b, "\n")
	}
	if t.Context != "" {
		fmt.Fprintf(&b, "**Context:** %s\n\n", t.Context)
	}

	fmt.Fprintf(&b, "## Workspace\n\n")
	fmt.Fprintf(&b, "Your workspace is at `workspace/` (symlinked to `.work/space/%s.%s/`).\n", goalBranch, t.ID)
	fmt.Fprintf(&b, "Use it for scratch files, intermediate outputs, and notes.\n\n")

	fmt.Fprintf(&b, "**Work Log:** `workspace/log.md`\n\n")

	fmt.Fprintf(&b, "## Cross-Task Context\n\n")
	fmt.Fprintf(&b, "You can read other tasks' workspaces via `.work/space/`.\n")

	return b.String()
}
