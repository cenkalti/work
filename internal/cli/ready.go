package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func readyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ready",
		Short: "List tasks with all dependencies met",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := filepath.Join(cwd, "workspace", "tasks")
			return runReady(tasksDir)
		},
	}
}

func runReady(tasksDir string) error {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("reading tasks: %w", err)
	}

	var ready []string
	for id, t := range tasks {
		if t.Status != task.StatusPending {
			continue
		}
		allMet := true
		for _, dep := range t.DependsOn {
			if d, ok := tasks[dep]; !ok || d.Status != task.StatusCompleted {
				allMet = false
				break
			}
		}
		if allMet {
			ready = append(ready, id)
		}
	}
	slices.Sort(ready)

	for _, id := range ready {
		fmt.Println(id)
	}
	return nil
}
