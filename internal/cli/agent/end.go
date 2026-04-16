package agent

import (
	"path/filepath"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/inbox"
	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

func endCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "end",
		Short:  "Mark agent session as ended (SessionEnd hook)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if loc, err := location.Detect(); err == nil {
				_ = inbox.Delete(filepath.Base(loc.RootRepo), loc.Branch)
			}
			existing, err := agent.Read(".")
			if err != nil {
				return nil
			}
			existing.Status = agent.StatusEnded
			return agent.Write(".", existing)
		},
	}
}
