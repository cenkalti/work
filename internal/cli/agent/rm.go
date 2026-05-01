package agent

import (
	"fmt"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/slot"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove an agent record (does not touch the worktree)",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names, _ := agentNames()
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			rec, err := findAgentByName(name)
			if err != nil {
				return err
			}
			if rec.CurrentSessionID != "" && agent.IsSessionRunning(rec.CurrentSessionID) {
				return fmt.Errorf("cannot remove %q: session %s is still running. Stop it first.", name, rec.CurrentSessionID)
			}
			if err := slot.ClearByUUID(rec.ID); err != nil {
				return fmt.Errorf("clearing slot: %w", err)
			}
			if err := agent.Delete(rec.ID); err != nil {
				return err
			}
			fmt.Printf("removed agent %s (%s)\n", name, rec.ID)
			return nil
		},
	}
}
