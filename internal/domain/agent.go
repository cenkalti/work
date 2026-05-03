package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// Agent is a managed Claude Code instance running in a Worktree. The on-disk
// record lives at ~/.work/agents/<UUID>.json. References to other domain
// types are stored as ID strings and resolved on demand via Repo, Worktree,
// Branch, Session methods.
type Agent struct {
	UUID string `json:"id"`
	Name string `json:"name"`

	RepoPath     string `json:"project_root"`
	WorktreeName string `json:"branch"`
	WorktreePath string `json:"worktree_path"`

	PaneID    string `json:"pane_id,omitempty"`
	TTYName   string `json:"tty_name,omitempty"`
	SessionID string `json:"current_session_id,omitempty"`

	Status            string `json:"status"`
	CurrentTool       string `json:"current_tool,omitempty"`
	NotificationCount int    `json:"notification_count,omitempty"`
	MessageCount      int    `json:"message_count,omitempty"`
	LastPromptPreview string `json:"last_prompt_preview,omitempty"`

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	StartedAt     time.Time `json:"started_at,omitzero"`
	LastActivity  time.Time `json:"last_activity,omitzero"`
	TurnStartedAt time.Time `json:"turn_started_at,omitzero"`

	Archived bool `json:"archived,omitempty"`
}

// Repo returns the agent's parent Repo.
func (a *Agent) Repo() Repo {
	return Repo{Path: a.RepoPath}
}

// Save atomically writes the agent record to disk. The directory is created
// if missing. As a side effect, identity + status fields are broadcast to
// WezTerm as user vars when TTYName is set.
func (a *Agent) Save() error {
	if a == nil || a.UUID == "" {
		return errors.New("agent.Save: uuid is empty")
	}
	dir, err := agentsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	final := filepath.Join(dir, a.UUID+".json")
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+a.UUID+".*.tmp")
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
	if a.TTYName != "" {
		wezterm.WriteUserVars(a.TTYName, a.userVars())
	}
	return nil
}

// Delete removes the agent record. Missing is not an error.
func (a *Agent) Delete() error {
	if a == nil || a.UUID == "" {
		return nil
	}
	dir, err := agentsDir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, a.UUID+".json")
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// userVars returns the subset of Agent fields broadcast to WezTerm as
// OSC SetUserVar values. Identity + status fields only.
func (a *Agent) userVars() map[string]string {
	taskID := BranchID(a.WorktreeName)
	if a.WorktreeName == "" {
		taskID = a.Repo().ProjectName()
	}
	return map[string]string{
		"agent_name":                a.Name,
		"agent_project":             a.Repo().ProjectName(),
		"agent_project_root":        a.RepoPath,
		"agent_task_id":             taskID,
		"agent_branch":              a.WorktreeName,
		"agent_worktree_path":       a.WorktreePath,
		"agent_pane_id":             a.PaneID,
		"agent_current_session_id":  a.SessionID,
		"agent_status":              a.Status,
		"agent_current_tool":        a.CurrentTool,
		"agent_notification_count":  strconv.Itoa(a.NotificationCount),
		"agent_message_count":       strconv.Itoa(a.MessageCount),
		"agent_last_prompt_preview": a.LastPromptPreview,
	}
}

// agentsDir returns ~/.work/agents/.
func agentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".work", "agents"), nil
}

// LoadAgent reads the agent record with the given UUID.
func LoadAgent(uuid string) (*Agent, error) {
	dir, err := agentsDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(dir, uuid+".json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var a Agent
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	return &a, nil
}

// ListAgents returns every agent record under ~/.work/agents/. Returns an
// empty slice if the directory does not exist.
func ListAgents() ([]*Agent, error) {
	dir, err := agentsDir()
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
	var out []*Agent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		// skip atomic-rename tempfiles
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		uuid := strings.TrimSuffix(e.Name(), ".json")
		a, err := LoadAgent(uuid)
		if err != nil {
			// tolerate a single bad file; skip it
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

// FindAgentByWorktree returns the agent whose stored WorktreePath matches
// path, or fs.ErrNotExist. Path comparison is on the resolved (symlink-evaluated)
// string the agent was created with.
func FindAgentByWorktree(path string) (*Agent, error) {
	all, err := ListAgents()
	if err != nil {
		return nil, err
	}
	for _, a := range all {
		if a.WorktreePath == path {
			return a, nil
		}
	}
	return nil, fs.ErrNotExist
}

// FindAgentBySession returns the agent whose current Session matches
// sessionID, or fs.ErrNotExist.
func FindAgentBySession(sessionID string) (*Agent, error) {
	if sessionID == "" {
		return nil, fs.ErrNotExist
	}
	all, err := ListAgents()
	if err != nil {
		return nil, err
	}
	for _, a := range all {
		if a.SessionID == sessionID {
			return a, nil
		}
	}
	return nil, fs.ErrNotExist
}
