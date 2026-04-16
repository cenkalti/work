package agent

import (
	"fmt"
	"path/filepath"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/inbox"
	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "status <running|idle>",
		Short:     "Update agent status (PreToolUse/Stop hook)",
		Hidden:    true,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{agent.StatusRunning, agent.StatusIdle},
		RunE: func(cmd *cobra.Command, args []string) error {
			status := args[0]
			if status != agent.StatusRunning && status != agent.StatusIdle {
				return fmt.Errorf("invalid status: %s", status)
			}
			existing, err := agent.Read(".")
			if err != nil {
				return nil
			}
			existing.Status = status
			if err := agent.Write(".", existing); err != nil {
				return err
			}
			if status == agent.StatusRunning {
				if loc, err := location.Detect(); err == nil {
					_ = inbox.Delete(filepath.Base(loc.RootRepo), loc.Branch)
				}
			}
			return nil
		},
	}
}
