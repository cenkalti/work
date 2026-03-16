package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/task"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func statusOrder(s string) int {
	switch s {
	case task.StatusActive:
		return 0
	case task.StatusPending:
		return 1
	case task.StatusCompleted:
		return 2
	default:
		return 3
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [goal]",
		Short: "List goals (from root) or tasks (from worktree or with arg)",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return listGoalWorktreeNames(detectLocation(cmd).RootRepo), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)

			// If explicit goal arg given, list its tasks
			if len(args) > 0 {
				return listTasks(paths.TasksDir(loc.RootRepo, args[0]))
			}

			// If in a goal/task worktree, list tasks
			if loc.Type == location.Goal || loc.Type == location.Task {
				return listTasks(loc.TasksDir())
			}

			// At root: list goals
			for _, name := range listGoalWorktreeNames(loc.RootRepo) {
				fmt.Println(name)
			}
			return nil
		},
	}
}

func listTasks(tasksDir string) error {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found; create tasks using the work MCP tool")
	}

	sorted := make([]*task.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status == "" {
			t.Status = task.StatusPending
		}
		sorted = append(sorted, t)
	}
	slices.SortFunc(sorted, func(a, b *task.Task) int {
		if c := statusOrder(a.Status) - statusOrder(b.Status); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	for _, t := range sorted {
		fmt.Printf("%-30s %s\n", t.ID, t.Status)
	}
	return nil
}
