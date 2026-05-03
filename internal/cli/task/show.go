package task

import (
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/domain"
	taskpkg "github.com/cenkalti/work/internal/task"
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
			tasksDir := domain.LocalTasksDir(cwd)
			return runShow(tasksDir, args[0])
		},
	}
}

func runShow(tasksDir, id string) error {
	t, err := taskpkg.Load(tasksDir, id)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(t)
	if err != nil {
		return err
	}

	fmt.Print(string(out))
	return nil
}
