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
	"github.com/cenkalti/work/internal/order"
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

	// Normalize the user-defined order: prune UUIDs no longer present, append
	// any newly-seen UUIDs at the end.
	ord, _ := order.Read()
	present := make(map[string]bool, len(rows))
	for _, r := range rows {
		present[r.AgentID] = true
	}
	cleaned := make([]string, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, u := range ord {
		if present[u] && !seen[u] {
			cleaned = append(cleaned, u)
			seen[u] = true
		}
	}
	for _, r := range rows {
		if !seen[r.AgentID] {
			cleaned = append(cleaned, r.AgentID)
			seen[r.AgentID] = true
		}
	}
	if !sliceEqual(cleaned, ord) {
		_ = order.Write(cleaned)
	}
	idx := make(map[string]int, len(cleaned))
	for i, u := range cleaned {
		idx[u] = i
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
		// both unassigned: by user-defined order
		return idx[ai.AgentID] < idx[aj.AgentID]
	})

	return rows, nil
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
