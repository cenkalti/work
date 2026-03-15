package agent

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/cenkalti/work/internal/task"
)

var goalTemplate = template.Must(template.New("goal").Parse(`# Work

You are working inside a goal worktree. Work is a multi-task orchestrator that decomposes plans into tasks with dependencies, then runs each task as a separate Claude Code instance in its own git worktree.

Use the ` + "`/work-plan`" + ` slash command to start the planning workflow.

## Workspace

Your workspace is at ` + "`.work/space/{{.GoalBranch}}/`" + `. Use it for all planning documents (goal.md, plan.md, research.md, etc.).

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

- ` + "`.work/space/{{.GoalBranch}}/goal.md`" + ` — The goal description
- ` + "`.work/space/{{.GoalBranch}}/plan.md`" + ` — The implementation plan (created during ` + "`/work-plan`" + `)
- ` + "`.work/space/{{.GoalBranch}}/tasks/`" + ` — Task JSON files (created by ` + "`work decompose`" + `)

## Cross-Goal Context

All goals share the ` + "`.work/space/`" + ` directory. You can read other goals' workspaces for context:
- ` + "`.work/space/<other-goal>/goal.md`" + `
- ` + "`.work/space/<other-goal>.<task-id>/log.md`" + `
`))

var taskTemplate = template.Must(template.New("task").Funcs(template.FuncMap{
	"join":      strings.Join,
	"trimSpace": strings.TrimSpace,
}).Parse(`{{- if .Goal -}}
# Goal

{{trimSpace .Goal}}

{{end -}}
# Your Task

**ID:** {{.Task.ID}}

**Summary:** {{.Task.TaskSummary}}

{{- if .Task.Description}}

**Description:** {{.Task.Description}}
{{- end}}
{{- if .Task.Files}}

**Files:** {{join .Task.Files ", "}}
{{- end}}
{{- if .Task.Acceptance}}

**Acceptance Criteria:**
{{- range .Task.Acceptance}}
- {{.}}
{{- end}}
{{end}}
{{- if .Task.Context}}
**Context:** {{.Task.Context}}

{{end -}}
## Workspace

Your workspace is at ` + "`workspace/`" + ` (symlinked to ` + "`.work/space/{{.GoalBranch}}.{{.Task.ID}}/`" + `).
Use it for scratch files, intermediate outputs, and notes.

**Work Log:** ` + "`workspace/log.md`" + `

## Cross-Task Context

You can read other tasks' workspaces via ` + "`.work/space/`" + `.
`))

type goalData struct {
	GoalBranch string
}

type taskData struct {
	GoalBranch string
	Goal       string
	Task       *task.Task
}

// GoalClaudeMD returns the CLAUDE.md content written into goal worktrees so Claude knows about work commands.
func GoalClaudeMD(goalBranch string) string {
	var buf bytes.Buffer
	if err := goalTemplate.Execute(&buf, goalData{GoalBranch: goalBranch}); err != nil {
		panic(err)
	}
	return buf.String()
}

// TaskClaudeMD returns the CLAUDE.md content written into task worktrees.
func TaskClaudeMD(goalBranch, goal string, t *task.Task) string {
	var buf bytes.Buffer
	if err := taskTemplate.Execute(&buf, taskData{GoalBranch: goalBranch, Goal: goal, Task: t}); err != nil {
		panic(err)
	}
	return buf.String()
}
