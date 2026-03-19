package task

import (
	"fmt"
	"os"
	"os/exec"
	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func editCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "edit <id>",
		Short:             "Edit a task in $EDITOR",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: taskIDCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				return fmt.Errorf("$EDITOR is not set")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			taskFile := taskpkg.File(paths.LocalTasksDir(cwd), args[0])
			if _, err := os.Stat(taskFile); err != nil {
				return fmt.Errorf("task %q not found", args[0])
			}
			c := exec.Command(editor, taskFile)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}
