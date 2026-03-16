package cli

import (
	"strings"
	"testing"

	"github.com/cenkalti/work/internal/task"
)

func TestRunTree_BasicOrder(t *testing.T) {
	tasks := []*task.Task{
		{ID: "root-task", Summary: "root"},
		{ID: "child-task", Summary: "child", DependsOn: []string{"root-task"}},
	}
	dir := writeTasks(t, tasks)

	out := captureStdout(t, func() {
		if err := runTree(dir, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "root-task") {
		t.Error("output missing root-task")
	}
	if !strings.Contains(out, "child-task") {
		t.Error("output missing child-task")
	}
	// The tree prints tasks with no dependents at the top; their deps appear below.
	// child-task has no dependents → it is the display root, printed first.
	// root-task is child-task's dep → printed below.
	posChild := strings.Index(out, "child-task")
	posRoot := strings.Index(out, "root-task")
	if posRoot < posChild {
		t.Error("root-task (a dependency) should appear below child-task in tree output")
	}
}

func TestRunTree_FilterByID(t *testing.T) {
	tasks := []*task.Task{
		{ID: "task-a", Summary: "A"},
		{ID: "task-b", Summary: "B", DependsOn: []string{"task-a"}},
		{ID: "task-c", Summary: "C"},
	}
	dir := writeTasks(t, tasks)

	out := captureStdout(t, func() {
		if err := runTree(dir, "task-b"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "task-b") {
		t.Error("output missing task-b")
	}
	if !strings.Contains(out, "task-a") {
		t.Error("output missing task-a (dep of task-b)")
	}
	if strings.Contains(out, "task-c") {
		t.Error("task-c should not appear (not in task-b subtree)")
	}
}

func TestRunTree_FilterNotFound(t *testing.T) {
	tasks := []*task.Task{
		{ID: "task-a", Summary: "A"},
	}
	dir := writeTasks(t, tasks)

	err := runTree(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown filter ID")
	}
}

func TestRunTree_CircularDependency(t *testing.T) {
	// task-top has no dependents (it's a root), and depends on task-a which circles back.
	tasks := []*task.Task{
		{ID: "task-top", Summary: "top", DependsOn: []string{"task-a"}},
		{ID: "task-a", Summary: "A", DependsOn: []string{"task-b"}},
		{ID: "task-b", Summary: "B", DependsOn: []string{"task-a"}},
	}
	dir := writeTasks(t, tasks)

	out := captureStdout(t, func() {
		runTree(dir, "")
	})

	if !strings.Contains(out, "circular") {
		t.Error("expected '(circular)' annotation in output")
	}
}

func TestRunTree_CompletedAnnotation(t *testing.T) {
	tasks := []*task.Task{
		{ID: "done-task", Summary: "done", Status: task.StatusCompleted},
	}
	dir := writeTasks(t, tasks)

	out := captureStdout(t, func() {
		runTree(dir, "")
	})

	if !strings.Contains(out, "completed") {
		t.Error("expected '(completed)' annotation for completed task")
	}
}
