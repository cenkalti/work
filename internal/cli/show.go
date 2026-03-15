package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show task details as YAML",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := workContext(cmd)
			goal, taskID, isTask := ctx.ResolveName(args[0])
			if !isTask {
				return fmt.Errorf("%q is not a task; use 'goal.task-id' or run from a goal worktree", args[0])
			}
			tasksDir := tasksDirFor(ctx.RootRepo, goal)
			return runShow(tasksDir, taskID)
		},
	}
}

func runShow(tasksDir, id string) error {
	data, err := os.ReadFile(filepath.Join(tasksDir, id+".json"))
	if err != nil {
		return fmt.Errorf("task %q not found", id)
	}

	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("parsing task: %w", err)
	}

	out, err := yaml.Marshal(t)
	if err != nil {
		return err
	}

	fmt.Print(string(out))
	return nil
}
