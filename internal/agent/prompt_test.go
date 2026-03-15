package agent

import (
	"strings"
	"testing"

	"github.com/cenkalti/work/internal/task"
)

func TestTaskClaudeMD(t *testing.T) {
	tk := &task.Task{
		ID:          "auth-impl",
		TaskSummary: "implement user authentication",
		Description: "Add JWT-based auth",
		Files:       []string{"auth.go", "middleware.go"},
		Acceptance:  []string{"login works", "tokens expire"},
	}
	prompt := TaskClaudeMD("kube-access", "optimize kubernetes performance", tk)

	checks := []string{
		"optimize kubernetes performance",
		"implement user authentication",
		"auth-impl",
		"Add JWT-based auth",
		"auth.go",
		"login works",
		"workspace/log.md",
		".work/space/kube-access.auth-impl/",
	}

	for _, want := range checks {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestGoalClaudeMD(t *testing.T) {
	md := GoalClaudeMD("update-deps")

	checks := []string{
		"/work-plan",
		".work/space/update-deps/",
		"work decompose",
		"work list",
	}

	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("GoalClaudeMD missing %q", want)
		}
	}
}
