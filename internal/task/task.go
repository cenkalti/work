package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusCompleted = "completed"
)

type Task struct {
	ID          string   `json:"id" yaml:"id"`
	TaskSummary string   `json:"task" yaml:"task"`
	DependsOn   []string `json:"depends_on" yaml:"depends_on"`
	Status      string   `json:"status" yaml:"status"`
	Files       []string `json:"files" yaml:"files"`
	Description string   `json:"description" yaml:"description"`
	Acceptance  []string `json:"acceptance" yaml:"acceptance"`
	Context     string   `json:"context,omitempty" yaml:"context,omitempty"`
}

// LoadAll reads all task JSON files from dir and returns them keyed by ID.
func LoadAll(dir string) (map[string]*Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tasks := make(map[string]*Task)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var t Task
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", e.Name(), err)
		}
		tasks[t.ID] = &t
	}
	return tasks, nil
}

func (t *Task) WriteToFile(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, t.ID+".json"), data, 0o644)
}
