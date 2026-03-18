package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

type workContextKey struct{}

func persistWorkContext(cmd *cobra.Command, args []string) error {
	wc, err := location.Detect()
	if err != nil {
		return err
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return nil
}

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

// taskIDCompletionFunc completes task IDs from ./workspace/tasks/ in the current directory.
func taskIDCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	tasksDir := filepath.Join(cwd, "workspace", "tasks")
	tasks, err := task.LoadAll(tasksDir)
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
