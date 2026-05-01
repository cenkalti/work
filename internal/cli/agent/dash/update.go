package dash

import (
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/slot"
	"github.com/cenkalti/work/internal/wezterm"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		m.LastRefresh = time.Time(msg)
		return m, tea.Batch(tickCmd(), loadRowsCmd())

	case rowsLoadedMsg:
		m.Rows = []Row(msg)
		if m.Cursor >= len(m.Rows) {
			m.Cursor = max(0, len(m.Rows)-1)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch s {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.Cursor+1 < len(m.Rows) {
			m.Cursor++
		}
		return m, nil
	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil
	case "enter":
		return m, m.jumpToCursor()
	case "alt+0":
		return m, m.unassignSlot()
	}

	// alt+1..9 assigns the cursor agent to that slot.
	if len(s) == 5 && s[:4] == "alt+" && s[4] >= '1' && s[4] <= '9' {
		return m, m.assignSlot(int(s[4] - '0'))
	}

	// bare 1..9 jumps to that slot's agent pane.
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		return m, m.jumpToSlot(int(s[0] - '0'))
	}

	return m, nil
}

func (m Model) jumpToSlot(n int) tea.Cmd {
	slots, err := slot.Read()
	if err != nil {
		return nil
	}
	uuid, ok := slots[n]
	if !ok || uuid == "" {
		return nil
	}
	return jumpToAgent(uuid)
}

func (m Model) jumpToCursor() tea.Cmd {
	if m.Cursor < 0 || m.Cursor >= len(m.Rows) {
		return nil
	}
	return jumpToAgent(m.Rows[m.Cursor].AgentID)
}

func jumpToAgent(uuid string) tea.Cmd {
	if uuid == "" {
		return nil
	}
	rec, err := agent.Read(uuid)
	if err != nil {
		return nil
	}
	if rec.PaneID == "" {
		return nil
	}
	return activatePaneCmd(rec.PaneID)
}

func (m Model) assignSlot(n int) tea.Cmd {
	if m.Cursor < 0 || m.Cursor >= len(m.Rows) {
		return nil
	}
	uuid := m.Rows[m.Cursor].AgentID
	if uuid == "" {
		return nil
	}
	_ = slot.Set(n, uuid)
	return loadRowsCmd()
}

func (m Model) unassignSlot() tea.Cmd {
	if m.Cursor < 0 || m.Cursor >= len(m.Rows) {
		return nil
	}
	uuid := m.Rows[m.Cursor].AgentID
	if uuid == "" {
		return nil
	}
	_ = slot.ClearByUUID(uuid)
	return loadRowsCmd()
}

// activatePaneCmd activates the target pane and raises its GUI window via the
// shared wezterm helper. Errors are silently ignored — a stale pane id is the
// expected case.
func activatePaneCmd(paneID string) tea.Cmd {
	return func() tea.Msg {
		_ = wezterm.ActivatePaneString(paneID)
		return nil
	}
}
