package task

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func statusOrder(s string) int {
	switch s {
	case taskpkg.StatusActive:
		return 0
	case taskpkg.StatusPending:
		return 1
	case taskpkg.StatusCompleted:
		return 2
	default:
		return 3
	}
}

func listCmd() *cobra.Command {
	var (
		flagReady     bool
		flagActive    bool
		flagBlocked   bool
		flagPending   bool
		flagCompleted bool
	)

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List subtasks",
		Long: `task ls              # list all subtasks
task ls --ready      # pending tasks with all dependencies met
task ls --active     # tasks currently being worked on
task ls --blocked    # pending tasks with unmet dependencies
task ls --pending    # all pending tasks
task ls --completed  # completed tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := paths.LocalTasksDir(cwd)
			return listTasks(tasksDir, flagReady, flagActive, flagBlocked, flagPending, flagCompleted)
		},
	}

	cmd.Flags().BoolVar(&flagReady, "ready", false, "pending tasks with all dependencies met")
	cmd.Flags().BoolVar(&flagActive, "active", false, "tasks currently being worked on")
	cmd.Flags().BoolVar(&flagBlocked, "blocked", false, "pending tasks with unmet dependencies")
	cmd.Flags().BoolVar(&flagPending, "pending", false, "all pending tasks")
	cmd.Flags().BoolVar(&flagCompleted, "completed", false, "completed tasks")

	return cmd
}

func matchesFilter(t *taskpkg.Task, all map[string]*taskpkg.Task, ready, active, blocked, pending, completed bool) bool {
	if active && t.Status == taskpkg.StatusActive {
		return true
	}
	if completed && t.Status == taskpkg.StatusCompleted {
		return true
	}
	if pending && t.Status == taskpkg.StatusPending {
		return true
	}
	if t.Status == taskpkg.StatusPending {
		depsOK := allDepsMet(t, all)
		if ready && depsOK {
			return true
		}
		if blocked && !depsOK {
			return true
		}
	}
	return false
}

func listTasks(tasksDir string, ready, active, blocked, pending, completed bool) error {
	tasks, err := taskpkg.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}
	if len(tasks) == 0 {
		return nil
	}

	noFilter := !ready && !active && !blocked && !pending && !completed

	var filtered []*taskpkg.Task
	for _, t := range tasks {
		if t.Status == "" {
			t.Status = taskpkg.StatusPending
		}
		if noFilter || matchesFilter(t, tasks, ready, active, blocked, pending, completed) {
			filtered = append(filtered, t)
		}
	}

	slices.SortFunc(filtered, func(a, b *taskpkg.Task) int {
		if c := statusOrder(a.Status) - statusOrder(b.Status); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	for _, t := range filtered {
		fmt.Printf("%-30s %s\n", t.ID, t.Status)
	}
	return nil
}

func allDepsMet(t *taskpkg.Task, all map[string]*taskpkg.Task) bool {
	for _, dep := range t.DependsOn {
		if d, ok := all[dep]; !ok || d.Status != taskpkg.StatusCompleted {
			return false
		}
	}
	return true
}
