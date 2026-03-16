package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/work/internal/task"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func completeCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "complete <name>",
		Short:             "Mark a task as completed",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			goal, taskID := loc.ResolveName(args[0])
			return runComplete(paths.TasksDir(loc.RootRepo, goal), taskID)
		},
	}
}

func runComplete(tasksDir, id string) error {
	taskFile := filepath.Join(tasksDir, id+".json")
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return fmt.Errorf("task %q not found", id)
	}

	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("parsing task: %w", err)
	}

	t.Status = task.StatusCompleted
	if err := t.WriteToFile(tasksDir); err != nil {
		return err
	}

	fmt.Printf("Marked %s as completed\n", id)
	return nil
}
