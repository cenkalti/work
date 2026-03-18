package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func statusOrder(s string) int {
	switch s {
	case task.StatusActive:
		return 0
	case task.StatusPending:
		return 1
	case task.StatusCompleted:
		return 2
	default:
		return 3
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tasks",
		Short: "List subtasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := filepath.Join(cwd, "workspace", "tasks")
			return listTasks(tasksDir)
		},
	}
}

func listTasks(tasksDir string) error {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found; create tasks using the work MCP tool")
	}

	sorted := make([]*task.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status == "" {
			t.Status = task.StatusPending
		}
		sorted = append(sorted, t)
	}
	slices.SortFunc(sorted, func(a, b *task.Task) int {
		if c := statusOrder(a.Status) - statusOrder(b.Status); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	for _, t := range sorted {
		fmt.Printf("%-30s %s\n", t.ID, t.Status)
	}
	return nil
}
