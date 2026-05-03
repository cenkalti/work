package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/inbox"
	"github.com/spf13/cobra"
)

const promptPreviewLen = 120

type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	ToolName      string `json:"tool_name"`
	Prompt        string `json:"prompt"`
	ToolInput     struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func hookCmd() *cobra.Command {
	root := &cobra.Command{
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
				return handleSessionEnd(&p)
			case "PreToolUse":
				return handlePreToolUse(&p)
			case "PostToolUse":
				return handlePostToolUse(&p)
			case "UserPromptSubmit":
				return handleUserPromptSubmit(&p)
			case "Stop", "PermissionRequest", "Elicitation":
				return handleStop(&p, 0)
			case "StopFailure":
				return handleStop(&p, 1)
			case "Notification":
				return handleNotification(&p)
			}
			return nil
		},
	}
	root.AddCommand(validateHTMLCmd())
	return root
}

// updateRecord finds the agent by session id and applies mutate. Missing record
// is a silent no-op.
func updateRecord(sessionID string, mutate func(*domain.Agent)) error {
	rec, err := domain.FindAgentBySession(sessionID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	mutate(rec)
	rec.UpdatedAt = time.Now().UTC()
	return rec.Save()
}

func handleSessionStart(p *hookPayload) error {
	if p.SessionID == "" {
		return fmt.Errorf("missing session_id")
	}
	writeOSC133("A")
	writeOSC133("B")
	now := time.Now().UTC()
	if err := updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusIdle
		r.StartedAt = now
		r.LastActivity = now
	}); err != nil {
		return err
	}
	repo, wt, err := domain.Detect()
	if err != nil {
		return err
	}
	if wt == nil || !isWorkManaged(repo) {
		return nil
	}
	return printTaskContext(*wt)
}

func handleSessionEnd(p *hookPayload) error {
	clearInbox(p.SessionID)
	now := time.Now().UTC()
	return updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusStopped
		r.CurrentTool = ""
		r.TurnStartedAt = time.Time{}
		r.NotificationCount = 0
		r.LastActivity = now
	})
}

func handlePreToolUse(p *hookPayload) error {
	now := time.Now().UTC()
	if err := updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusToolRunning
		r.CurrentTool = p.ToolName
		r.NotificationCount = 0
		r.LastActivity = now
	}); err != nil {
		return err
	}
	clearInbox(p.SessionID)
	if p.ToolName == "Bash" {
		return runBashCheck(p.ToolInput.Command, os.Stdout)
	}
	return nil
}

func handlePostToolUse(p *hookPayload) error {
	now := time.Now().UTC()
	return updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusRunning
		r.CurrentTool = ""
		r.NotificationCount = 0
		r.LastActivity = now
	})
}

func handleUserPromptSubmit(p *hookPayload) error {
	writeOSC133("C")
	now := time.Now().UTC()
	if err := updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusRunning
		r.LastPromptPreview = truncatePrompt(p.Prompt, promptPreviewLen)
		r.MessageCount++
		r.NotificationCount = 0
		r.TurnStartedAt = now
		r.LastActivity = now
	}); err != nil {
		return err
	}
	clearInbox(p.SessionID)
	return nil
}

func handleStop(p *hookPayload, exit int) error {
	writeOSC133(fmt.Sprintf("D;%d", exit))
	writeOSC133("A")
	writeOSC133("B")
	now := time.Now().UTC()
	return updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusIdle
		r.CurrentTool = ""
		r.TurnStartedAt = time.Time{}
		r.LastActivity = now
	})
}

func handleNotification(p *hookPayload) error {
	now := time.Now().UTC()
	if err := updateRecord(p.SessionID, func(r *domain.Agent) {
		r.Status = domain.StatusAwaitingInput
		r.NotificationCount++
		r.LastActivity = now
	}); err != nil {
		return err
	}
	repo, _, err := domain.Detect()
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
	project := repo.ProjectName()
	subpath := ""
	if resolved != repo.Path {
		if wtRootResolved, err := filepath.EvalSymlinks(repo.WorktreeRoot()); err == nil {
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

func clearInbox(sessionID string) {
	if sessionID == "" {
		return
	}
	_ = inbox.Delete(sessionID)
}

func truncatePrompt(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

