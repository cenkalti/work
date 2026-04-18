package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
			case "Stop", "StopFailure", "PermissionRequest", "Elicitation":
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
	// Elicitation responses don't fire a hook, so PreToolUse is the first
	// signal we get that Claude has resumed working after a prompt.
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
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		resolved = cwd
	}
	project := filepath.Base(loc.RootRepo)
	subpath := ""
	if resolved != loc.RootRepo {
		wtRoot := filepath.Join(loc.RootRepo, ".work", "tree")
		if wtRootResolved, err := filepath.EvalSymlinks(wtRoot); err == nil {
			if rel, ok := strings.CutPrefix(resolved, wtRootResolved+string(filepath.Separator)); ok {
				subpath = rel
			}
		}
	}
	return inbox.Write(&inbox.Message{
		Project:   project,
		Branch:    subpath,
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
	state, err := agentpkg.Read(".")
	if err != nil {
		return
	}
	_ = inbox.Delete(state.ID)
}
