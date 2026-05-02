// Package dash is the Bubbletea TUI for `agent dash`.
package dash

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"github.com/sahilm/fuzzy"
)

// Row is one rendered agent line in the dashboard.
//
// Populated in data.go. Empty during the skeleton.
type Row struct {
	AgentID            string
	WorktreePath       string
	Slot               int  // 0 = unassigned
	HasSlot            bool
	HasNotification    bool
	Status             string
	Project            string
	Name               string
	Branch             string
	Session            string
	CurrentTool        string
	TurnElapsed        time.Duration
	HasTurnElapsed     bool
	LastActivity       time.Time
	HasLastActivity    bool
	LastPromptPreview  string
	NoWorktree         bool
	Crashed            bool
	Attached           bool
	Dirty              bool
	HasTask            bool
	TasksCompleted     int
	TasksTotal         int
	TodoIDs            []string
}

// Model holds the TUI state.
type Model struct {
	Rows         []Row
	Cursor       int
	Width        int
	Height       int
	LastRefresh  time.Time
	ShowArchived bool
	Filter       textinput.Model
	Filtering    bool
	Quit         bool
}

// NewModel returns an empty model.
func NewModel() Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "filter"
	ti.SetVirtualCursor(true)
	return Model{Filter: ti}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), loadRowsCmd(m.ShowArchived))
}

// visibleRows returns the rows currently shown after applying the fuzzy
// filter, if any. Order is fuzzy-score-descending when filtering.
func (m Model) visibleRows() []Row {
	q := m.Filter.Value()
	if q == "" {
		return m.Rows
	}
	keys := make([]string, len(m.Rows))
	for i, r := range m.Rows {
		keys[i] = r.Project + "/" + r.Name
	}
	matches := fuzzy.Find(q, keys)
	out := make([]Row, 0, len(matches))
	for _, mt := range matches {
		out = append(out, m.Rows[mt.Index])
	}
	return out
}
