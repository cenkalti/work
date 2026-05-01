package dash

import (
	"strconv"
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/order"
	"github.com/cenkalti/work/internal/slot"
	"github.com/cenkalti/work/internal/wezterm"
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tickMsg:
		m.LastRefresh = time.Time(msg)
		return m, tea.Batch(tickCmd(), loadRowsCmd(m.ShowArchived))

	case rowsLoadedMsg:
		m.Rows = []Row(msg)
		if m.Cursor >= len(m.Rows) {
			m.Cursor = max(0, len(m.Rows)-1)
		}
		return m, loadDirtyCmd(m.Rows)

	case dirtyLoadedMsg:
		for i := range m.Rows {
			if d, ok := msg[m.Rows[i].AgentID]; ok {
				m.Rows[i].Dirty = d
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "J", "shift+down":
		return m.moveDown()
	case "K", "shift+up":
		return m.moveUp()
	case "enter":
		return m, m.jumpToCursor()
	case "alt+0":
		return m, m.unassignSlot()
	case "a":
		if m.ShowArchived {
			return m, nil
		}
		return m, m.archiveCursor()
	case "u":
		if !m.ShowArchived {
			return m, nil
		}
		return m, m.unarchiveCursor()
	case "A":
		m.ShowArchived = !m.ShowArchived
		m.Cursor = 0
		return m, loadRowsCmd(m.ShowArchived)
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
	return func() tea.Msg {
		// If a pane is recorded and still alive, jump to it.
		if rec.PaneID != "" {
			if err := wezterm.ActivatePaneString(rec.PaneID); err == nil {
				if id, err := strconv.Atoi(rec.PaneID); err == nil {
					wezterm.MaximizePane(id)
				}
				return nil
			}
		}
		// Otherwise spawn a new window in the worktree running `agent run`.
		if id, err := agent.SpawnRunWindow(rec.WorktreePath); err == nil {
			wezterm.MaximizePane(id)
		}
		return nil
	}
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
	return loadRowsCmd(m.ShowArchived)
}

// moveUp swaps the cursor row with the row immediately above it in the user
// order, then moves the cursor to follow it.
func (m Model) moveUp() (tea.Model, tea.Cmd) {
	if m.Cursor <= 0 || m.Cursor >= len(m.Rows) {
		return m, nil
	}
	a := m.Rows[m.Cursor].AgentID
	b := m.Rows[m.Cursor-1].AgentID
	if changed, _ := order.Swap(a, b); !changed {
		return m, nil
	}
	m.Rows[m.Cursor], m.Rows[m.Cursor-1] = m.Rows[m.Cursor-1], m.Rows[m.Cursor]
	m.Cursor--
	return m, loadRowsCmd(m.ShowArchived)
}

// moveDown swaps the cursor row with the row immediately below it in the user
// order, then moves the cursor to follow it.
func (m Model) moveDown() (tea.Model, tea.Cmd) {
	if m.Cursor < 0 || m.Cursor+1 >= len(m.Rows) {
		return m, nil
	}
	a := m.Rows[m.Cursor].AgentID
	b := m.Rows[m.Cursor+1].AgentID
	if changed, _ := order.Swap(a, b); !changed {
		return m, nil
	}
	m.Rows[m.Cursor], m.Rows[m.Cursor+1] = m.Rows[m.Cursor+1], m.Rows[m.Cursor]
	m.Cursor++
	return m, loadRowsCmd(m.ShowArchived)
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
	return loadRowsCmd(m.ShowArchived)
}

func (m Model) archiveCursor() tea.Cmd {
	if m.Cursor < 0 || m.Cursor >= len(m.Rows) {
		return nil
	}
	uuid := m.Rows[m.Cursor].AgentID
	if uuid == "" {
		return nil
	}
	rec, err := agent.Read(uuid)
	if err != nil {
		return nil
	}
	rec.Archived = true
	_ = agent.Write(rec)
	_ = slot.ClearByUUID(uuid)
	return loadRowsCmd(m.ShowArchived)
}

func (m Model) unarchiveCursor() tea.Cmd {
	if m.Cursor < 0 || m.Cursor >= len(m.Rows) {
		return nil
	}
	uuid := m.Rows[m.Cursor].AgentID
	if uuid == "" {
		return nil
	}
	rec, err := agent.Read(uuid)
	if err != nil {
		return nil
	}
	rec.Archived = false
	_ = agent.Write(rec)
	return loadRowsCmd(m.ShowArchived)
}

