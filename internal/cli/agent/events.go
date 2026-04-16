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

type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func hookCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook",
		Short:  "Dispatch Claude Code hook events (reads event from stdin)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			var p hookPayload
			if err := json.Unmarshal(data, &p); err != nil {
				return err
			}
			switch p.HookEventName {
			case "SessionStart":
				return handleSessionStart(&p)
			case "SessionEnd":
				return handleSessionEnd()
			case "PreToolUse":
				return handlePreToolUse(&p)
			case "UserPromptSubmit":
				return handleUserPromptSubmit()
			case "Stop":
				return handleStop()
			case "Notification":
				return handleNotification(&p)
			}
			return nil
		},
	}
}

func handleSessionStart(p *hookPayload) error {
	if p.SessionID == "" {
		return fmt.Errorf("missing session_id")
	}
	existing, err := agentpkg.Read(".")
	if err == nil && existing.ID != p.SessionID && existing.Status != agentpkg.StatusEnded {
		if agentpkg.IsSessionRunning(existing.ID) {
			return fmt.Errorf("another session is already running: %s", existing.ID)
		}
	}
	if err := agentpkg.Write(".", &agentpkg.State{
		ID:     p.SessionID,
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
}

func handleSessionEnd() error {
	clearInbox()
	existing, err := agentpkg.Read(".")
	if err != nil {
		return nil
	}
	existing.Status = agentpkg.StatusEnded
	return agentpkg.Write(".", existing)
}

func handlePreToolUse(p *hookPayload) error {
	setAgentStatus(agentpkg.StatusRunning)
	clearInbox()
	if p.ToolName == "Bash" {
		return runBashCheck(p.ToolInput.Command, os.Stdout)
	}
	return nil
}

func handleUserPromptSubmit() error {
	setAgentStatus(agentpkg.StatusRunning)
	clearInbox()
	return nil
}

func handleStop() error {
	setAgentStatus(agentpkg.StatusIdle)
	return nil
}

func handleNotification(p *hookPayload) error {
	setAgentStatus(agentpkg.StatusIdle)
	loc, err := location.Detect()
	if err != nil {
		return err
	}
	return inbox.Write(&inbox.Message{
		Project:   filepath.Base(loc.RootRepo),
		Branch:    loc.Branch,
		SessionID: p.SessionID,
		Timestamp: time.Now(),
	})
}

func setAgentStatus(status string) {
	existing, err := agentpkg.Read(".")
	if err != nil {
		return
	}
	existing.Status = status
	_ = agentpkg.Write(".", existing)
}

func clearInbox() {
	loc, err := location.Detect()
	if err != nil {
		return
	}
	_ = inbox.Delete(filepath.Base(loc.RootRepo), loc.Branch)
}
