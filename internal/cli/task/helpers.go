package task

import (
	"os"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

// taskIDCompletionFunc completes task IDs from ./workspace/tasks/ in the current directory.
func taskIDCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	tasksDir := paths.LocalTasksDir(cwd)
	tasks, err := taskpkg.LoadAll(tasksDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var ids []string
	for id := range tasks {
		if strings.HasPrefix(id, toComplete) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, cobra.ShellCompDirectiveNoFileComp
}
