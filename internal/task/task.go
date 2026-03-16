package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validID = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusCompleted = "completed"
)

type Task struct {
	ID          string   `json:"id" yaml:"id"`
	Summary     string   `json:"summary" yaml:"summary"`
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

// DetectCycle checks for circular dependencies in the given task map.
// Returns an error describing the cycle if one is found.
func DetectCycle(tasks map[string]*Task) error {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(id string, path []string) error
	dfs = func(id string, path []string) error {
		if inStack[id] {
			cycle := append(path, id)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " → "))
		}
		if visited[id] {
			return nil
		}
		visited[id] = true
		inStack[id] = true
		t, ok := tasks[id]
		if ok {
			for _, dep := range t.DependsOn {
				if err := dfs(dep, append(path, id)); err != nil {
					return err
				}
			}
		}
		inStack[id] = false
		return nil
	}

	for id := range tasks {
		if err := dfs(id, nil); err != nil {
			return err
		}
	}
	return nil
}

func (t *Task) WriteToFile(dir string) error {
	if !validID.MatchString(t.ID) {
		return fmt.Errorf("invalid task ID %q: must be kebab-case (lowercase alphanumeric and hyphens)", t.ID)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, t.ID+".json"), data, 0o644)
}
