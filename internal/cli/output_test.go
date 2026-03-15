package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cenkalti/work/internal/task"
)

// writeTasks writes a slice of tasks to a temp dir and returns the dir path.
func writeTasks(t *testing.T, tasks []*task.Task) string {
	t.Helper()
	dir := t.TempDir()
	for _, tk := range tasks {
		if err := tk.WriteToFile(dir); err != nil {
			t.Fatalf("writing task %s: %v", tk.ID, err)
		}
	}
	return dir
}

// captureStdout runs f and returns everything written to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

var testTasks = []*task.Task{
	{ID: "task-a", TaskSummary: "A", Status: task.StatusPending},
	{ID: "task-b", TaskSummary: "B", Status: task.StatusActive},
	{ID: "task-c", TaskSummary: "C", Status: task.StatusCompleted},
	{ID: "task-d", TaskSummary: "D", Status: task.StatusPending, DependsOn: []string{"task-a"}},
	{ID: "task-e", TaskSummary: "E", Status: task.StatusPending, DependsOn: []string{"task-c"}},
}

func TestListTasks(t *testing.T) {
	dir := writeTasks(t, testTasks)

	out := captureStdout(t, func() {
		if err := listTasks(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, id := range []string{"task-a", "task-b", "task-c", "task-d", "task-e"} {
		if !strings.Contains(out, id) {
			t.Errorf("output missing %q", id)
		}
	}
	// active should appear before pending
	posB := strings.Index(out, "task-b")
	posA := strings.Index(out, "task-a")
	if posB > posA {
		t.Error("active task-b should be listed before pending task-a")
	}
}

func TestListTasks_Empty(t *testing.T) {
	dir := t.TempDir()
	err := listTasks(dir)
	if err == nil {
		t.Error("expected error for empty task dir")
	}
}

func TestListTasks_MissingDir(t *testing.T) {
	err := listTasks(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Error("expected error for missing dir")
	}
}

func TestRunReady(t *testing.T) {
	dir := writeTasks(t, testTasks)

	out := captureStdout(t, func() {
		if err := runReady(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	lines := strings.Fields(out)
	// task-a: pending, no deps → ready
	// task-e: pending, dep task-c is completed → ready
	// task-d: pending, dep task-a is pending → NOT ready
	// task-b: active → NOT ready
	if !strings.Contains(out, "task-a") {
		t.Error("task-a should be ready (no deps)")
	}
	if !strings.Contains(out, "task-e") {
		t.Error("task-e should be ready (dep completed)")
	}
	for _, l := range lines {
		if l == "task-d" {
			t.Error("task-d should not be ready (dep pending)")
		}
		if l == "task-b" {
			t.Error("task-b should not be ready (active, not pending)")
		}
	}
}

func TestRunReady_AllComplete(t *testing.T) {
	tasks := []*task.Task{
		{ID: "task-a", TaskSummary: "A", Status: task.StatusCompleted},
	}
	dir := writeTasks(t, tasks)
	out := captureStdout(t, func() { runReady(dir) })
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output when all tasks complete, got %q", out)
	}
}

func TestActiveCmd(t *testing.T) {
	dir := writeTasks(t, testTasks)

	tasks, err := task.LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	var active []string
	for id, tk := range tasks {
		if tk.Status == task.StatusActive {
			active = append(active, id)
		}
	}

	out := captureStdout(t, func() {
		for _, id := range active {
			fmt.Println(id)
		}
	})

	if !strings.Contains(out, "task-b") {
		t.Error("task-b should appear in active output")
	}
	if strings.Contains(out, "task-a") {
		t.Error("task-a (pending) should not appear in active output")
	}
}
