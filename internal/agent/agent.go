package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/work/internal/wezterm"
)

const (
	StatusRunning       = "running"
	StatusIdle          = "idle"
	StatusToolRunning   = "tool_running"
	StatusAwaitingInput = "awaiting_input"
	StatusStopped       = "stopped"
)

// Record is the central agent record stored at ~/.work/agents/<id>.json.
type Record struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	Project      string `json:"project"`
	ProjectRoot  string `json:"project_root"`
	TaskID       string `json:"task_id"`
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktree_path"`

	PaneID           string `json:"pane_id,omitempty"`
	TTYName          string `json:"tty_name,omitempty"`
	CurrentSessionID string `json:"current_session_id,omitempty"`

	Status             string `json:"status"`
	CurrentTool        string `json:"current_tool,omitempty"`
	NotificationCount  int    `json:"notification_count,omitempty"`
	MessageCount       int    `json:"message_count,omitempty"`
	LastPromptPreview  string `json:"last_prompt_preview,omitempty"`

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	StartedAt     time.Time `json:"started_at,omitzero"`
	LastActivity  time.Time `json:"last_activity,omitzero"`
	TurnStartedAt time.Time `json:"turn_started_at,omitzero"`

	Archived bool `json:"archived,omitempty"`
}

// Dir returns the central agents directory: ~/.work/agents/.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".work", "agents"), nil
}

// Path returns the full path to an agent record by ID.
func Path(id string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// Read returns the agent record with the given ID.
func Read(id string) (*Record, error) {
	p, err := Path(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	return &r, nil
}

// Write atomically writes the agent record. The directory is created if missing.
func Write(r *Record) error {
	if r == nil || r.ID == "" {
		return errors.New("agent.Write: record id is empty")
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	final := filepath.Join(dir, r.ID+".json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+r.ID+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, final); err != nil {
		return err
	}
	if r.TTYName != "" {
		wezterm.WriteUserVars(r.TTYName, recordUserVars(r))
	}
	return nil
}

// recordUserVars returns the subset of Record fields broadcast to WezTerm as
// OSC SetUserVar values. Identity + status fields only; timestamps and the
// internal id/tty_name/archived fields are skipped.
func recordUserVars(r *Record) map[string]string {
	return map[string]string{
		"agent_name":                r.Name,
		"agent_project":             r.Project,
		"agent_project_root":        r.ProjectRoot,
		"agent_task_id":             r.TaskID,
		"agent_branch":              r.Branch,
		"agent_worktree_path":       r.WorktreePath,
		"agent_pane_id":             r.PaneID,
		"agent_current_session_id":  r.CurrentSessionID,
		"agent_status":              r.Status,
		"agent_current_tool":        r.CurrentTool,
		"agent_notification_count":  strconv.Itoa(r.NotificationCount),
		"agent_message_count":       strconv.Itoa(r.MessageCount),
		"agent_last_prompt_preview": r.LastPromptPreview,
	}
}

// Delete removes the agent record file. Missing is not an error.
func Delete(id string) error {
	p, err := Path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// List returns every agent record under ~/.work/agents/. Returns an empty slice
// if the directory does not exist.
func List() ([]*Record, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		// skip our atomic-rename tempfiles
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		r, err := Read(id)
		if err != nil {
			// tolerate a single bad file; skip it
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// FindByWorktree returns the record with WorktreePath == path, or fs.ErrNotExist.
func FindByWorktree(path string) (*Record, error) {
	all, err := List()
	if err != nil {
		return nil, err
	}
	for _, r := range all {
		if r.WorktreePath == path {
			return r, nil
		}
	}
	return nil, fs.ErrNotExist
}

// FindBySession returns the record with CurrentSessionID == sessionID, or fs.ErrNotExist.
func FindBySession(sessionID string) (*Record, error) {
	if sessionID == "" {
		return nil, fs.ErrNotExist
	}
	all, err := List()
	if err != nil {
		return nil, err
	}
	for _, r := range all {
		if r.CurrentSessionID == sessionID {
			return r, nil
		}
	}
	return nil, fs.ErrNotExist
}

// IsSessionRunning checks if a claude process with the given session ID is running.
func IsSessionRunning(sessionID string) bool {
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || filepath.Base(fields[0]) != "claude" {
			continue
		}
		for i := 1; i < len(fields)-1; i++ {
			if (fields[i] == "--resume" || fields[i] == "--session-id") && strings.EqualFold(fields[i+1], sessionID) {
				return true
			}
		}
	}
	return false
}

// RunningSessionIDs returns the set of session IDs from running claude processes.
func RunningSessionIDs() map[string]struct{} {
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return nil
	}
	ids := make(map[string]struct{})
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || filepath.Base(fields[0]) != "claude" {
			continue
		}
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "--resume" || fields[i] == "--session-id" {
				ids[strings.ToLower(fields[i+1])] = struct{}{}
				break
			}
		}
	}
	return ids
}

// HookInput is the common JSON structure received on stdin from Claude Code hooks.
type HookInput struct {
	SessionID string `json:"session_id"`
}

// ReadHookInput reads and parses the hook JSON from stdin.
func ReadHookInput() (*HookInput, error) {
	data, err := os.ReadFile("/dev/stdin")
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}
	if input.SessionID == "" {
		return nil, fmt.Errorf("no session_id in hook input")
	}
	return &input, nil
}
