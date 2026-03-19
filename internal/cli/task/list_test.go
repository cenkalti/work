package task

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taskpkg "github.com/cenkalti/work/internal/task"
)

// writeTasks writes a slice of tasks to a temp dir and returns the dir path.
func writeTasks(t *testing.T, tasks []*taskpkg.Task) string {
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

var testTasks = []*taskpkg.Task{
	{ID: "task-a", Summary: "A", Status: taskpkg.StatusPending},
	{ID: "task-b", Summary: "B", Status: taskpkg.StatusActive},
	{ID: "task-c", Summary: "C", Status: taskpkg.StatusCompleted},
	{ID: "task-d", Summary: "D", Status: taskpkg.StatusPending, DependsOn: []string{"task-a"}},
	{ID: "task-e", Summary: "E", Status: taskpkg.StatusPending, DependsOn: []string{"task-c"}},
}

func TestListTasks(t *testing.T) {
	dir := writeTasks(t, testTasks)

	out := captureStdout(t, func() {
		if err := listTasks(dir, false, false, false, false, false); err != nil {
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
	err := listTasks(dir, false, false, false, false, false)
	if err != nil {
		t.Errorf("unexpected error for empty task dir: %v", err)
	}
}

func TestListTasks_MissingDir(t *testing.T) {
	err := listTasks(filepath.Join(t.TempDir(), "nonexistent"), false, false, false, false, false)
	if err == nil {
		t.Error("expected error for missing dir")
	}
}

func loadTestTasks(t *testing.T) map[string]*taskpkg.Task {
	t.Helper()
	m := make(map[string]*taskpkg.Task)
	for _, tk := range testTasks {
		m[tk.ID] = tk
	}
	return m
}

func TestAllDepsMet(t *testing.T) {
	all := loadTestTasks(t)

	// task-a: no deps -> met
	if !allDepsMet(all["task-a"], all) {
		t.Error("task-a should have all deps met (no deps)")
	}
	// task-e: depends on task-c (completed) -> met
	if !allDepsMet(all["task-e"], all) {
		t.Error("task-e should have all deps met (task-c completed)")
	}
	// task-d: depends on task-a (pending) -> not met
	if allDepsMet(all["task-d"], all) {
		t.Error("task-d should not have all deps met (task-a pending)")
	}
}

func TestMatchesFilter_Ready(t *testing.T) {
	all := loadTestTasks(t)

	if !matchesFilter(all["task-a"], all, true, false, false, false, false) {
		t.Error("task-a should be ready (pending, no deps)")
	}
	if !matchesFilter(all["task-e"], all, true, false, false, false, false) {
		t.Error("task-e should be ready (pending, dep completed)")
	}
	if matchesFilter(all["task-d"], all, true, false, false, false, false) {
		t.Error("task-d should not be ready (dep pending)")
	}
	if matchesFilter(all["task-b"], all, true, false, false, false, false) {
		t.Error("task-b should not be ready (active)")
	}
}

func TestMatchesFilter_Blocked(t *testing.T) {
	all := loadTestTasks(t)

	if !matchesFilter(all["task-d"], all, false, false, true, false, false) {
		t.Error("task-d should be blocked (dep pending)")
	}
	if matchesFilter(all["task-a"], all, false, false, true, false, false) {
		t.Error("task-a should not be blocked (no deps)")
	}
}

func TestMatchesFilter_Active(t *testing.T) {
	all := loadTestTasks(t)

	if !matchesFilter(all["task-b"], all, false, true, false, false, false) {
		t.Error("task-b should match active filter")
	}
	if matchesFilter(all["task-a"], all, false, true, false, false, false) {
		t.Error("task-a should not match active filter")
	}
}

func TestRunSet(t *testing.T) {
	dir := writeTasks(t, []*taskpkg.Task{
		{ID: "my-task", Summary: "test", Status: taskpkg.StatusPending},
	})

	if err := runSet(dir, "my-task", taskpkg.StatusCompleted); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tasks, err := taskpkg.LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if tasks["my-task"].Status != taskpkg.StatusCompleted {
		t.Errorf("expected completed, got %s", tasks["my-task"].Status)
	}
}
