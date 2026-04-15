package agent

import (
	"github.com/cenkalti/work/internal/agent"
	"github.com/spf13/cobra"
)

func endCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "end",
		Short:  "Mark agent session as ended (SessionEnd hook)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			existing, err := agent.Read(".")
			if err != nil {
				return nil
			}
			existing.Status = agent.StatusEnded
			return agent.Write(".", existing)
		},
	}
}
