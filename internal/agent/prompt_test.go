package agent

import (
	"strings"
	"testing"

	"github.com/cenkalti/work/internal/task"
)

func TestTaskClaudeMD(t *testing.T) {
	tk := &task.Task{
		ID:          "auth-impl",
		Summary:     "implement user authentication",
		Description: "Add JWT-based auth",
		Files:       []string{"auth.go", "middleware.go"},
		Acceptance:  []string{"login works", "tokens expire"},
	}
	prompt := TaskClaudeMD("kube-access.auth-impl", "optimize kubernetes performance", tk)

	checks := []string{
		"optimize kubernetes performance",
		"implement user authentication",
		"auth-impl",
		"Add JWT-based auth",
		"auth.go",
		"login works",
		"workspace/log.md",
	}

	for _, want := range checks {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}

	// Optional fields absent when empty
	tkMin := &task.Task{ID: "min", Summary: "minimal task"}
	minPrompt := TaskClaudeMD("parent.min", "", tkMin)
	if strings.Contains(minPrompt, "# Parent Task") {
		t.Error("expected no Parent Task section when parentContext is empty")
	}
	if strings.Contains(minPrompt, "Description") {
		t.Error("expected no Description when empty")
	}
	if strings.Contains(minPrompt, "Acceptance Criteria") {
		t.Error("expected no Acceptance Criteria when empty")
	}
}

func TestRootTaskClaudeMD(t *testing.T) {
	md := RootTaskClaudeMD("update-deps")

	checks := []string{
		"/work-plan",
		"workspace/plan.md",
		"workspace/tasks/",
		"work ls",
	}

	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("RootTaskClaudeMD missing %q", want)
		}
	}

	if !strings.Contains(md, "update-deps") {
		t.Error("RootTaskClaudeMD should contain the branch name")
	}
}
