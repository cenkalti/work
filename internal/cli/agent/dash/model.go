// Package dash is the Bubbletea TUI for `agent dash`.
package dash

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	Rows        []Row
	Cursor      int
	Width       int
	Height      int
	LastRefresh time.Time
	Quit        bool
}

// NewModel returns an empty model.
func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), loadRowsCmd())
}
