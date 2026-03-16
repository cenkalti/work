package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
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
			branch := loc.ResolveName(args[0])
			parent := paths.ParentBranch(branch)
			if parent == "" {
				return fmt.Errorf("%q is a root task and has no task file", args[0])
			}
			taskID := paths.BranchID(branch)
			return runComplete(paths.TasksDir(loc.RootRepo, parent), taskID)
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
