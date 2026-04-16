package task

import (
	"fmt"
	"os"
	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "rm <id>",
		Short:             "Remove a task",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: taskIDCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := paths.LocalTasksDir(cwd)
			id := args[0]
			taskFile := taskpkg.File(tasksDir, id)
			if err := os.Remove(taskFile); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("task %q not found", id)
				}
				return fmt.Errorf("removing task %q: %w", id, err)
			}
			fmt.Printf("Removed task %s\n", id)
			return nil
		},
	}
}
