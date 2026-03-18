package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

var validStatuses = []string{task.StatusPending, task.StatusActive, task.StatusCompleted}

func setStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-status <id> <status>",
		Short: "Set task status",
		Long: `work set-status <id> pending     # mark task as pending
work set-status <id> active      # mark task as active
work set-status <id> completed   # mark task as completed`,
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
			tasksDir := filepath.Join(cwd, "workspace", "tasks")
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
	taskFile := filepath.Join(tasksDir, id+".json")
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return fmt.Errorf("task %q not found", id)
	}

	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("parsing task: %w", err)
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
