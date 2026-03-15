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

	// Optional fields absent when empty
	tkMin := &task.Task{ID: "min", TaskSummary: "minimal task"}
	minPrompt := TaskClaudeMD("goal", "", tkMin)
	if strings.Contains(minPrompt, "# Goal") {
		t.Error("expected no Goal section when goal is empty")
	}
	if strings.Contains(minPrompt, "Description") {
		t.Error("expected no Description when empty")
	}
	if strings.Contains(minPrompt, "Acceptance Criteria") {
		t.Error("expected no Acceptance Criteria when empty")
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

	// Goal branch is interpolated, not a different branch
	if strings.Contains(md, "update-deps/goal.md") && !strings.Contains(md, ".work/space/update-deps/goal.md") {
		t.Error("goal.md path not correctly prefixed")
	}
}
