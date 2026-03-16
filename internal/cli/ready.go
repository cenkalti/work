package cli

import (
	"fmt"
	"slices"

	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func readyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ready [task]",
		Short: "List tasks with all dependencies met",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return listRootTaskNames(detectLocation(cmd).RootRepo), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			var explicit string
			if len(args) > 0 {
				explicit = args[0]
			}
			branch, err := loc.ResolveBranch(explicit)
			if err != nil {
				return err
			}
			return runReady(paths.TasksDir(loc.RootRepo, branch))
		},
	}
}

func runReady(tasksDir string) error {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("reading tasks: %w", err)
	}

	var ready []string
	for id, t := range tasks {
		if t.Status != task.StatusPending {
			continue
		}
		allMet := true
		for _, dep := range t.DependsOn {
			if d, ok := tasks[dep]; !ok || d.Status != task.StatusCompleted {
				allMet = false
				break
			}
		}
		if allMet {
			ready = append(ready, id)
		}
	}
	slices.Sort(ready)

	for _, id := range ready {
		fmt.Println(id)
	}
	return nil
}
