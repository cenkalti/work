package cli

import (
	"fmt"
	"slices"

	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/task"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func treeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree [goal] [task-id]",
		Short: "Show tasks in a dependency tree",
		Args:  cobra.MaximumNArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			loc := detectLocation(cmd)
			switch len(args) {
			case 0:
				if loc.Type == location.Goal || loc.Type == location.Task {
					return listTaskIDsFiltered(loc.TasksDir(), nil), cobra.ShellCompDirectiveNoFileComp
				}
				return listGoalWorktreeNames(loc.RootRepo), cobra.ShellCompDirectiveNoFileComp
			case 1:
				if loc.Type == location.Root {
					return listTaskIDsFiltered(paths.TasksDir(loc.RootRepo, args[0]), nil), cobra.ShellCompDirectiveNoFileComp
				}
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)

			var goal, filterID string
			switch {
			case len(args) == 2:
				goal = args[0]
				filterID = args[1]
			case len(args) == 1 && (loc.Type == location.Goal || loc.Type == location.Task):
				goal = loc.Goal
				filterID = args[0]
			case len(args) == 1:
				goal = args[0]
			default:
				var err error
				goal, err = loc.ResolveGoal("")
				if err != nil {
					return err
				}
			}

			return runTree(paths.TasksDir(loc.RootRepo, goal), filterID)
		},
	}
}

func runTree(tasksDir, filterID string) error {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("reading tasks: %w", err)
	}

	depOf := make(map[string]bool)
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			depOf[dep] = true
		}
	}

	if filterID != "" {
		if _, ok := tasks[filterID]; !ok {
			return fmt.Errorf("task %q not found", filterID)
		}
		printTree(filterID, tasks, "", true, true, nil)
		return nil
	}

	var roots []string
	for id := range tasks {
		if !depOf[id] {
			roots = append(roots, id)
		}
	}
	slices.Sort(roots)

	for i, root := range roots {
		printTree(root, tasks, "", i == len(roots)-1, true, nil)
	}

	return nil
}

func printTree(id string, tasks map[string]*task.Task, prefix string, last bool, isRoot bool, visited map[string]bool) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	var connector string
	if isRoot {
		connector = ""
	} else if last {
		connector = "└─"
	} else {
		connector = "├─"
	}

	if visited[id] {
		fmt.Printf("%s%s%s (circular)\n", prefix, connector, id)
		return
	}

	label := id
	if t, ok := tasks[id]; ok && t.Status == task.StatusCompleted {
		label += " (completed)"
	}
	fmt.Printf("%s%s%s\n", prefix, connector, label)

	t, ok := tasks[id]
	if !ok {
		return
	}

	visited[id] = true
	defer func() { visited[id] = false }()

	deps := make([]string, len(t.DependsOn))
	copy(deps, t.DependsOn)
	slices.Sort(deps)

	var childPrefix string
	if isRoot {
		childPrefix = prefix
	} else if last {
		childPrefix = prefix + "  "
	} else {
		childPrefix = prefix + "│ "
	}

	for i, dep := range deps {
		printTree(dep, tasks, childPrefix, i == len(deps)-1, false, visited)
	}
}
