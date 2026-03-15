package task

import (
	"os"
	"testing"
)

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
