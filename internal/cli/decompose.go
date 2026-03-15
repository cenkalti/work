package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/claude"
	"github.com/spf13/cobra"
)

func decomposeCmd() *cobra.Command {
	var instructions string

	cmd := &cobra.Command{
		Use:   "decompose [goal]",
		Short: "Decompose a plan file into individual task JSON files",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return listGoalWorktreeNames(workContext(cmd).RootRepo), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := workContext(cmd)
			var explicit string
			if len(args) > 0 {
				explicit = args[0]
			}
			goal, err := ctx.ResolveGoal(explicit)
			if err != nil {
				return err
			}
			spacePath := goalSpacePath(ctx.RootRepo, goal)

			data, err := os.ReadFile(filepath.Join(spacePath, "plan.md"))
			if err != nil {
				return fmt.Errorf("reading plan file: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Decomposing plan into tasks...\n")

			tasks, err := claude.ExtractTasks(context.Background(), string(data), instructions)
			if err != nil {
				return err
			}

			tasksDir := filepath.Join(spacePath, "tasks")
			for _, t := range tasks {
				if err := t.WriteToFile(tasksDir); err != nil {
					return fmt.Errorf("writing task %s: %w", t.ID, err)
				}
			}

			fmt.Printf("\nCreated %d tasks:\n\n", len(tasks))
			for _, t := range tasks {
				deps := "none"
				if len(t.DependsOn) > 0 {
					deps = strings.Join(t.DependsOn, ", ")
				}
				fmt.Printf("  %-30s  depends on: %s\n", t.ID, deps)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&instructions, "instructions", "i", "", "additional instructions for task decomposition")
	return cmd
}
