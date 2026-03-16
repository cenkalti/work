package task

import (
	"os"
	"strings"
	"testing"
)

func TestDetectCycle(t *testing.T) {
	tasks := func(pairs ...string) map[string]*Task {
		m := make(map[string]*Task)
		for i := 0; i < len(pairs); i += 2 {
			id := pairs[i]
			deps := []string{}
			if pairs[i+1] != "" {
				deps = strings.Split(pairs[i+1], ",")
			}
			m[id] = &Task{ID: id, DependsOn: deps}
		}
		return m
	}

	if err := DetectCycle(tasks("a", "", "b", "a", "c", "b")); err != nil {
		t.Errorf("linear chain should have no cycle: %v", err)
	}
	if err := DetectCycle(tasks("a", "b", "b", "a")); err == nil {
		t.Error("direct cycle should be detected")
	}
	if err := DetectCycle(tasks("a", "b", "b", "c", "c", "a")); err == nil {
		t.Error("indirect cycle should be detected")
	}
	if err := DetectCycle(tasks("a", "b,c", "b", "", "c", "")); err != nil {
		t.Errorf("diamond shape should have no cycle: %v", err)
	}
	if err := DetectCycle(make(map[string]*Task)); err != nil {
		t.Errorf("empty map should have no cycle: %v", err)
	}
}

func TestWriteToFile_IDValidation(t *testing.T) {
	dir := t.TempDir()

	valid := []string{
		"my-task",
		"fix-auth",
		"task1",
		"a",
		"abc-123-def",
	}
	for _, id := range valid {
		tk := &Task{ID: id, TaskSummary: "test"}
		if err := tk.WriteToFile(dir); err != nil {
			t.Errorf("valid ID %q rejected: %v", id, err)
		}
		os.Remove(dir + "/" + id + ".json")
	}

	invalid := []string{
		"",
		"../evil",
		"path/traversal",
		"Has Spaces",
		"UPPER",
		"has.dot",
		"-leading-hyphen",
		"trailing-hyphen-",
	}
	for _, id := range invalid {
		tk := &Task{ID: id, TaskSummary: "test"}
		if err := tk.WriteToFile(dir); err == nil {
			t.Errorf("invalid ID %q should have been rejected", id)
		}
	}
}
