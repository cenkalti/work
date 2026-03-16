package agent

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/cenkalti/work/internal/task"
)

//go:embed root.md.tmpl
var rootTmpl string

//go:embed task.md.tmpl
var taskTmpl string

var rootTemplate = template.Must(template.New("root").Parse(rootTmpl))

var taskTemplate = template.Must(template.New("task").Funcs(template.FuncMap{
	"join":      strings.Join,
	"trimSpace": strings.TrimSpace,
}).Parse(taskTmpl))

type rootData struct {
	Branch string
}

type taskData struct {
	Branch        string
	ParentContext string
	Task          *task.Task
}

// RootTaskClaudeMD returns the CLAUDE.md content for root task worktrees.
func RootTaskClaudeMD(branch string) string {
	var buf bytes.Buffer
	if err := rootTemplate.Execute(&buf, rootData{Branch: branch}); err != nil {
		panic(fmt.Sprintf("bug: template execution failed: %v", err))
	}
	return buf.String()
}

// TaskClaudeMD returns the CLAUDE.md content for child task worktrees.
func TaskClaudeMD(branch, parentContext string, t *task.Task) string {
	var buf bytes.Buffer
	if err := taskTemplate.Execute(&buf, taskData{Branch: branch, ParentContext: parentContext, Task: t}); err != nil {
		panic(fmt.Sprintf("bug: template execution failed: %v", err))
	}
	return buf.String()
}
