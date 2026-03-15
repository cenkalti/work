package cli

import (
	"fmt"
	"slices"

	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func activeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "active [goal]",
		Short: "List tasks currently being worked on",
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
			tasksDir := tasksDirFor(ctx.RootRepo, goal)
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
