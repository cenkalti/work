package dash

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/slot"
	tea "github.com/charmbracelet/bubbletea"
)

// claudeSession is a parsed ~/.claude/sessions/<pid>.json.
type claudeSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

type rowsLoadedMsg []Row

// loadRowsCmd asynchronously loads rows from disk.
func loadRowsCmd() tea.Cmd {
	return func() tea.Msg {
		rows, _ := loadRows()
		return rowsLoadedMsg(rows)
	}
}

func loadRows() ([]Row, error) {
	recs, err := agent.List()
	if err != nil {
		return nil, err
	}
	slots, err := slot.Read()
	if err != nil {
		return nil, err
	}
	sessions, err := readClaudeSessions()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(recs))
	for _, r := range recs {
		row := Row{
			AgentID:           r.ID,
			Status:            r.Status,
			Project:           r.Project,
			Name:              r.Name,
			CurrentTool:       r.CurrentTool,
			HasNotification:   r.NotificationCount > 0,
			LastPromptPreview: r.LastPromptPreview,
		}
		if !r.LastActivity.IsZero() {
			row.LastActivity = r.LastActivity
			row.HasLastActivity = true
		}
		if !r.TurnStartedAt.IsZero() {
			row.TurnElapsed = time.Since(r.TurnStartedAt)
			row.HasTurnElapsed = true
		}
		// Slot lookup by UUID.
		for k, v := range slots {
			if v == r.ID {
				row.Slot = k
				row.HasSlot = true
				break
			}
		}
		// Liveness vs. crashed: record claims a running-style status but no
		// matching session file is present.
		if r.CurrentSessionID != "" {
			_, alive := sessions[r.CurrentSessionID]
			row.Attached = alive
			if !alive && (r.Status == agent.StatusRunning || r.Status == agent.StatusToolRunning || r.Status == agent.StatusAwaitingInput) {
				row.Crashed = true
			}
		}
		// Worktree existence.
		if r.WorktreePath != "" {
			if _, err := os.Stat(r.WorktreePath); errors.Is(err, fs.ErrNotExist) {
				row.NoWorktree = true
			}
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		ai, aj := rows[i], rows[j]
		switch {
		case ai.HasSlot && !aj.HasSlot:
			return true
		case !ai.HasSlot && aj.HasSlot:
			return false
		case ai.HasSlot && aj.HasSlot:
			return ai.Slot < aj.Slot
		}
		// both unassigned: by last_activity desc
		switch {
		case ai.HasLastActivity && !aj.HasLastActivity:
			return true
		case !ai.HasLastActivity && aj.HasLastActivity:
			return false
		case ai.HasLastActivity && aj.HasLastActivity:
			return ai.LastActivity.After(aj.LastActivity)
		}
		// fallback: by name
		return strings.ToLower(ai.Name) < strings.ToLower(aj.Name)
	})

	return rows, nil
}

func readClaudeSessions() (map[string]claudeSession, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".claude", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]claudeSession{}, nil
		}
		return nil, err
	}
	out := make(map[string]claudeSession, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s claudeSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if s.SessionID != "" {
			out[s.SessionID] = s
		}
	}
	return out, nil
}
