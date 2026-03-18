package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/task"
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
	var (
		flagReady     bool
		flagActive    bool
		flagBlocked   bool
		flagPending   bool
		flagCompleted bool
	)

	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List subtasks",
		Long: `work tasks              # list all subtasks
work tasks --ready      # pending tasks with all dependencies met
work tasks --active     # tasks currently being worked on
work tasks --blocked    # pending tasks with unmet dependencies
work tasks --pending    # all pending tasks
work tasks --completed  # completed tasks`,
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

func matchesFilter(t *task.Task, all map[string]*task.Task, ready, active, blocked, pending, completed bool) bool {
	if active && t.Status == task.StatusActive {
		return true
	}
	if completed && t.Status == task.StatusCompleted {
		return true
	}
	if pending && t.Status == task.StatusPending {
		return true
	}
	if t.Status == task.StatusPending {
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
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found; create tasks using the work MCP tool")
	}

	noFilter := !ready && !active && !blocked && !pending && !completed

	var filtered []*task.Task
	for _, t := range tasks {
		if t.Status == "" {
			t.Status = task.StatusPending
		}
		if noFilter || matchesFilter(t, tasks, ready, active, blocked, pending, completed) {
			filtered = append(filtered, t)
		}
	}

	slices.SortFunc(filtered, func(a, b *task.Task) int {
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

func allDepsMet(t *task.Task, all map[string]*task.Task) bool {
	for _, dep := range t.DependsOn {
		if d, ok := all[dep]; !ok || d.Status != task.StatusCompleted {
			return false
		}
	}
	return true
}
