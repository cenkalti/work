package agent

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/cenkalti/work/internal/task"
)

//go:embed goal.md.tmpl
var goalTmpl string

//go:embed task.md.tmpl
var taskTmpl string

var goalTemplate = template.Must(template.New("goal").Parse(goalTmpl))

var taskTemplate = template.Must(template.New("task").Funcs(template.FuncMap{
	"join":      strings.Join,
	"trimSpace": strings.TrimSpace,
}).Parse(taskTmpl))

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
