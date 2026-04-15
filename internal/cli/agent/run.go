package agent

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/cenkalti/work/internal/agent"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start or resume a claude session",
		RunE: func(cmd *cobra.Command, args []string) error {
			claudeBin, err := exec.LookPath("claude")
			if err != nil {
				return err
			}
			existing, err := agent.Read(".")
			if err == nil {
				return syscall.Exec(claudeBin, []string{"claude", "--resume", existing.ID}, os.Environ())
			}
			id := uuid.New().String()
			if err := agent.Write(".", &agent.State{ID: id, Status: agent.StatusRunning}); err != nil {
				return err
			}
			return syscall.Exec(claudeBin, []string{"claude", "--session-id", id}, os.Environ())
		},
	}
}
