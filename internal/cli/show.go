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
		Use:               "show <id>",
		Short:             "Show task details as YAML",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: taskIDCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := filepath.Join(cwd, "workspace", "tasks")
			return runShow(tasksDir, args[0])
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
