package task

import (
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

var validStatuses = []string{taskpkg.StatusPending, taskpkg.StatusActive, taskpkg.StatusCompleted}

func setStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-status <id> <status>",
		Short: "Set task status",
		Long: `task set-status <id> pending     # mark task as pending
task set-status <id> active      # mark task as active
task set-status <id> completed   # mark task as completed`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: setCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, status := args[0], args[1]
			if !isValidStatus(status) {
				return fmt.Errorf("invalid status %q; must be one of: pending, active, completed", status)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := paths.LocalTasksDir(cwd)
			return runSet(tasksDir, id, status)
		},
	}
}

func isValidStatus(s string) bool {
	for _, v := range validStatuses {
		if s == v {
			return true
		}
	}
	return false
}

func runSet(tasksDir, id, status string) error {
	t, err := taskpkg.Load(tasksDir, id)
	if err != nil {
		return err
	}

	t.Status = status
	if err := t.WriteToFile(tasksDir); err != nil {
		return err
	}

	fmt.Printf("Set %s to %s\n", id, status)
	return nil
}

func setCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		// Complete task IDs.
		return taskIDCompletionFunc(cmd, args, toComplete)
	case 1:
		// Complete statuses.
		return validStatuses, cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
