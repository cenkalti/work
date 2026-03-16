package cli

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/task"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

type workContextKey struct{}

// persistWorkContext calls Detect and stores the result in the command's context.
func persistWorkContext(cmd *cobra.Command, args []string) error {
	wc, err := location.Detect()
	if err != nil {
		return err
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return nil
}

// detectLocation retrieves the *WorkContext stored by PersistentPreRunE.
// If not yet stored (e.g. during shell completion), it detects and caches it.
// Exits the program if context detection fails.
func detectLocation(cmd *cobra.Command) *location.Location {
	if wc, ok := cmd.Context().Value(workContextKey{}).(*location.Location); ok {
		return wc
	}
	wc, err := location.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return wc
}

// listGoalWorktreeNames returns worktree names that are goals (no dots).
func listGoalWorktreeNames(rootRepo string) []string {
	var goals []string
	wtRoot := paths.WorktreeRoot(rootRepo)
	for _, name := range listWorktreeNames(wtRoot) {
		if !strings.Contains(name, ".") {
			goals = append(goals, name)
		}
	}
	return goals
}

// listTaskIDsFiltered returns task IDs matching an optional filter.
func listTaskIDsFiltered(tasksDir string, filter func(*task.Task) bool) []string {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return nil
	}
	var ids []string
	for id, t := range tasks {
		if filter == nil || filter(t) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}
