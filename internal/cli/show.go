package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "show <name>",
		Short:             "Show task details as YAML",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			branch := loc.ResolveName(args[0])
			parent := paths.ParentBranch(branch)
			if parent == "" {
				return fmt.Errorf("%q is a root task and has no task file; use 'task.subtask' notation or run from a task worktree", args[0])
			}
			taskID := paths.BranchID(branch)
			return runShow(paths.TasksDir(loc.RootRepo, parent), taskID)
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
