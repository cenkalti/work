package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func activeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "active",
		Short: "List tasks currently being worked on",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			tasksDir := filepath.Join(cwd, "workspace", "tasks")
			tasks, err := task.LoadAll(tasksDir)
			if err != nil {
				return fmt.Errorf("reading tasks: %w", err)
			}
			var active []string
			for id, t := range tasks {
				if t.Status == task.StatusActive {
					active = append(active, id)
				}
			}
			slices.Sort(active)
			for _, id := range active {
				fmt.Println(id)
			}
			return nil
		},
	}
}
