package task

import (
	"fmt"
	"os"
	"slices"

	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func treeCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "tree [id]",
		Short:             "Show tasks in a dependency tree",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: taskIDCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := paths.LocalTasksDir(cwd)
			var filterID string
			if len(args) > 0 {
				filterID = args[0]
			}
			return runTree(tasksDir, filterID)
		},
	}
}

func runTree(tasksDir, filterID string) error {
	tasks, err := taskpkg.LoadAll(tasksDir)
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

func printTree(id string, tasks map[string]*taskpkg.Task, prefix string, last bool, isRoot bool, visited map[string]bool) {
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
	if t, ok := tasks[id]; ok && t.Status == taskpkg.StatusCompleted {
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
