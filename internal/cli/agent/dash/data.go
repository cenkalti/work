package dash

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/order"
	"github.com/cenkalti/work/internal/slot"
	"github.com/cenkalti/work/internal/task"
	todopkg "github.com/cenkalti/work/internal/todo"
	tea "charm.land/bubbletea/v2"
)

// claudeSession is a parsed ~/.claude/sessions/<pid>.json.
type claudeSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

type rowsLoadedMsg []Row

// dirtyLoadedMsg maps agent ID -> dirty state.
type dirtyLoadedMsg map[string]bool

// loadRowsCmd asynchronously loads rows from disk.
func loadRowsCmd(showArchived bool) tea.Cmd {
	return func() tea.Msg {
		rows, _ := loadRows(showArchived)
		return rowsLoadedMsg(rows)
	}
}

// dirtyTTL is how long a dirty-status result stays valid in the cache before
// we re-shell out to git for it.
const dirtyTTL = 10 * time.Second

// dirtyConcurrency caps the number of `git status` processes running at once
// to avoid CPU spikes when many worktrees expire together.
const dirtyConcurrency = 2

type dirtyEntry struct {
	dirty bool
	at    time.Time
}

var (
	dirtyCacheMu sync.Mutex
	dirtyCache   = map[string]dirtyEntry{}
)

// dirtyCachePeek returns the last known dirty value for path regardless of
// freshness. Used to seed rows so the indicator does not blink while a refresh
// is in flight.
func dirtyCachePeek(path string) (bool, bool) {
	dirtyCacheMu.Lock()
	defer dirtyCacheMu.Unlock()
	e, ok := dirtyCache[path]
	return e.dirty, ok
}

func dirtyCacheFresh(path string) bool {
	dirtyCacheMu.Lock()
	defer dirtyCacheMu.Unlock()
	e, ok := dirtyCache[path]
	return ok && time.Since(e.at) < dirtyTTL
}

func dirtyCachePut(path string, dirty bool) {
	dirtyCacheMu.Lock()
	defer dirtyCacheMu.Unlock()
	dirtyCache[path] = dirtyEntry{dirty: dirty, at: time.Now()}
}

// loadDirtyCmd computes dirty state for rows whose cache entry is missing or
// stale, throttled to dirtyConcurrency parallel git invocations. Fresh entries
// are skipped; rows already carry the cached value from loadRows.
func loadDirtyCmd(rows []Row) tea.Cmd {
	type entry struct {
		id, path string
	}
	entries := make([]entry, 0, len(rows))
	for _, r := range rows {
		if r.NoWorktree || r.WorktreePath == "" {
			continue
		}
		if dirtyCacheFresh(r.WorktreePath) {
			continue
		}
		entries = append(entries, entry{id: r.AgentID, path: r.WorktreePath})
	}
	return func() tea.Msg {
		result := make(map[string]bool, len(entries))
		if len(entries) == 0 {
			return dirtyLoadedMsg(result)
		}
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, dirtyConcurrency)
		for _, e := range entries {
			wg.Add(1)
			go func(id, path string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				d := git.IsDirty(path)
				dirtyCachePut(path, d)
				mu.Lock()
				result[id] = d
				mu.Unlock()
			}(e.id, e.path)
		}
		wg.Wait()
		return dirtyLoadedMsg(result)
	}
}

func loadRows(showArchived bool) ([]Row, error) {
	recs, err := domain.ListAgents()
	if err != nil {
		return nil, err
	}
	filtered := recs[:0]
	for _, r := range recs {
		if r.Archived == showArchived {
			filtered = append(filtered, r)
		}
	}
	recs = filtered
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
			AgentID:           r.UUID,
			WorktreePath:      r.WorktreePath,
			Status:            r.Status,
			Project:           r.Repo().ProjectName(),
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
			if v == r.UUID {
				row.Slot = k
				row.HasSlot = true
				break
			}
		}
		// Liveness vs. crashed: record claims a running-style status but no
		// matching session file is present.
		if r.SessionID != "" {
			_, alive := sessions[r.SessionID]
			row.Attached = alive
			if !alive && (r.Status == domain.StatusRunning || r.Status == domain.StatusToolRunning || r.Status == domain.StatusAwaitingInput) {
				row.Crashed = true
			}
			row.Session = claudeSessionName(r.WorktreePath, r.SessionID)
		}
		// Worktree existence.
		if r.WorktreePath != "" {
			if _, err := os.Stat(r.WorktreePath); errors.Is(err, fs.ErrNotExist) {
				row.NoWorktree = true
			}
		}
		// Current branch, with WORK_BRANCH_PREFIX-style prefixes stripped
		// for display: anything before the first "/" is dropped so
		// e.g. "jakealti/kube-url-routing" shows as "kube-url-routing".
		// This is lossy for legitimately namespaced branches like
		// "release/v1.2" (would show "v1.2"); revisit by capturing the
		// prefix into the agent record if that bites. Empty on detached
		// HEAD or git error.
		if !row.NoWorktree && r.WorktreePath != "" {
			if b := git.CurrentBranch(r.WorktreePath); b != "" {
				if i := strings.Index(b, "/"); i >= 0 {
					b = b[i+1:]
				}
				row.Branch = b
			}
		}
		// Seed dirty from the last cached value (regardless of freshness) so
		// the indicator does not blink while loadDirtyCmd refreshes it.
		if !row.NoWorktree && row.WorktreePath != "" {
			if d, ok := dirtyCachePeek(row.WorktreePath); ok {
				row.Dirty = d
			}
		}
		if c, t, ok := taskProgress(r.RepoPath, r.WorktreeName); ok {
			row.HasTask = true
			row.TasksCompleted = c
			row.TasksTotal = t
		}
		rows = append(rows, row)
	}

	// Match todos to agents by their & handles. A handle of the form
	// "<project>/<name>" links the todo to the agent with that project and
	// name. Plain "<project>" handles (no slash) are ignored here.
	if todoIndex, err := loadTodoAgentIndex(); err == nil {
		for i := range rows {
			key := rows[i].Project + "/" + rows[i].Name
			if ids := todoIndex[key]; len(ids) > 0 {
				rows[i].TodoIDs = ids
			}
		}
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

// loadTodoAgentIndex scans every open todo for & handles of the form
// "<project>/<name>" and returns map[<project>/<name>] -> []todoID.
// Closed (completed/cancelled) todos are excluded.
func loadTodoAgentIndex() (map[string][]string, error) {
	dir, err := todopkg.Dir()
	if err != nil {
		return nil, err
	}
	todos, _, err := todopkg.LoadAll(dir)
	if err != nil {
		return nil, err
	}
	idx := make(map[string][]string, len(todos))
	for id, t := range todos {
		if todopkg.IsClosed(t.Status) {
			continue
		}
		for _, h := range t.Projects {
			if !strings.Contains(h, "/") {
				continue
			}
			idx[h] = append(idx[h], id)
		}
	}
	return idx, nil
}

// taskProgress reports the completed/total task counts for an agent.
//
// Parent agent (own task has subtasks): aggregate of subtasks in the agent's
// own workspace tasks dir.
// Leaf agent (own task has no subtasks): 0/1 or 1/1 from the agent's own task
// file, which lives in the parent branch's workspace.
// No task associated (root branch with no children, or empty branch): returns
// ok=false so the cell is rendered empty.
func taskProgress(root, branch string) (completed, total int, ok bool) {
	if root == "" || branch == "" {
		return 0, 0, false
	}
	wt := domain.Worktree{RepoPath: root, Name: branch}
	if subs, err := task.LoadAll(wt.TasksDir()); err == nil && len(subs) > 0 {
		for _, t := range subs {
			if t.Status == task.StatusCompleted {
				completed++
			}
		}
		return completed, len(subs), true
	}
	parent := domain.ParentBranchName(branch)
	if parent == "" {
		return 0, 0, false
	}
	parentWt := domain.Worktree{RepoPath: root, Name: parent}
	t, err := task.Load(parentWt.TasksDir(), domain.BranchID(branch))
	if err != nil {
		return 0, 0, false
	}
	if t.Status == task.StatusCompleted {
		return 1, 1, true
	}
	return 0, 1, true
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
