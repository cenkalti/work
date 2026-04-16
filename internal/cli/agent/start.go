package agent

import (
	"fmt"

	"github.com/cenkalti/work/internal/agent"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "start",
		Short:  "Register agent session (SessionStart hook)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := agent.ReadHookInput()
			if err != nil {
				return err
			}
			existing, err := agent.Read(".")
			if err == nil && existing.ID != input.SessionID && existing.Status != agent.StatusEnded {
				if agent.IsSessionRunning(existing.ID) {
					return fmt.Errorf("another session is already running: %s", existing.ID)
				}
			}
			return agent.Write(".", &agent.State{
				ID:     input.SessionID,
				Status: agent.StatusIdle,
			})
		},
	}
}
