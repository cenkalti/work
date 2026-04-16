package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	agentpkg "github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/inbox"
	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

func sessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "session-start",
		Short:  "Handle SessionStart hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := agentpkg.ReadHookInput()
			if err != nil {
				return err
			}
			existing, err := agentpkg.Read(".")
			if err == nil && existing.ID != input.SessionID && existing.Status != agentpkg.StatusEnded {
				if agentpkg.IsSessionRunning(existing.ID) {
					return fmt.Errorf("another session is already running: %s", existing.ID)
				}
			}
			if err := agentpkg.Write(".", &agentpkg.State{
				ID:     input.SessionID,
				Status: agentpkg.StatusIdle,
			}); err != nil {
				return err
			}
			loc, err := location.Detect()
			if err != nil {
				return err
			}
			if loc.IsRoot() || !isWorkManaged(loc.RootRepo) {
				return nil
			}
			return printTaskContext(loc.RootRepo, loc.Branch)
		},
	}
}

func sessionEndCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "session-end",
		Short:  "Handle SessionEnd hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if loc, err := location.Detect(); err == nil {
				_ = inbox.Delete(filepath.Base(loc.RootRepo), loc.Branch)
			}
			existing, err := agentpkg.Read(".")
			if err != nil {
				return nil
			}
			existing.Status = agentpkg.StatusEnded
			return agentpkg.Write(".", existing)
		},
	}
}

func preToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "pre-tool-use",
		Short:  "Handle PreToolUse hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			setAgentStatus(agentpkg.StatusRunning)
			clearInbox()

			var hi hookInput
			_ = json.Unmarshal(data, &hi)
			if hi.ToolName == "Bash" {
				return runBashCheck(hi.ToolInput.Command, os.Stdout)
			}
			return nil
		},
	}
}

func userPromptSubmitCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "user-prompt-submit",
		Short:  "Handle UserPromptSubmit hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setAgentStatus(agentpkg.StatusRunning)
			clearInbox()
			return nil
		},
	}
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "stop",
		Short:  "Handle Stop hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setAgentStatus(agentpkg.StatusIdle)
			return nil
		},
	}
}

func notificationCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "notification",
		Short:  "Handle Notification hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := agentpkg.ReadHookInput()
			if err != nil {
				return err
			}
			setAgentStatus(agentpkg.StatusIdle)
			loc, err := location.Detect()
			if err != nil {
				return err
			}
			return inbox.Write(&inbox.Message{
				Project:   filepath.Base(loc.RootRepo),
				Branch:    loc.Branch,
				SessionID: input.SessionID,
				Timestamp: time.Now(),
			})
		},
	}
}

// setAgentStatus updates the .agent file status. Silent no-op if no .agent file.
func setAgentStatus(status string) {
	existing, err := agentpkg.Read(".")
	if err != nil {
		return
	}
	existing.Status = status
	_ = agentpkg.Write(".", existing)
}

// clearInbox removes the current agent's inbox message, if any.
func clearInbox() {
	loc, err := location.Detect()
	if err != nil {
		return
	}
	_ = inbox.Delete(filepath.Base(loc.RootRepo), loc.Branch)
}
