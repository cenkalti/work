package dash

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/cenkalti/work/internal/agent"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	cellStyle     = lipgloss.NewStyle().PaddingRight(1)
	cursorStyle   = lipgloss.NewStyle().Reverse(true).PaddingRight(1)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	notifStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	attachedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dirtyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	statusStyles = map[string]lipgloss.Style{
		agent.StatusRunning:       lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		agent.StatusToolRunning:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		agent.StatusAwaitingInput: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),
		agent.StatusIdle:          lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		agent.StatusStopped:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}

	footerStyle = lipgloss.NewStyle().Faint(true)
)

const (
	colN = iota
	colProject
	colName
	colNotif
	colAttach
	colDirty
	colStatus
	colTasks
	colTodo
	colTool
	colTurn
	colLast
	colPrompt
)

var headers = []string{
	"N", "PROJECT", "NAME", "!", "C", "D",
	"STATUS", "TASKS", "TODO", "TOOL", "TURN", "LAST", "PROMPT",
}

func (m Model) View() tea.View {
	body := m.renderTable()

	var b strings.Builder
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteByte('\n')
	}

	if len(m.Rows) == 0 {
		b.WriteString(dimStyle.Render("(no agents — run `agent run` in a worktree to register one)"))
		b.WriteByte('\n')
	}

	if m.Height > 0 {
		used := strings.Count(body, "\n") + 1
		if len(m.Rows) == 0 {
			used++
		}
		footerHeight := 2
		blank := m.Height - used - footerHeight
		for range blank {
			b.WriteByte('\n')
		}
	}

	b.WriteString(footerStyle.Render(footerLine(m)))
	b.WriteByte('\n')
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m Model) renderTable() string {
	rows := make([][]string, 0, len(m.Rows))
	for _, r := range m.Rows {
		rows = append(rows, rowCells(r))
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(false).
		Headers(headers...).
		Rows(rows...).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.PaddingRight(1)
			}
			if row == m.Cursor && col <= colName {
				return cursorStyle
			}
			return cellStyle
		})

	if m.Width > 0 {
		t.Width(m.Width)
	}
	return t.Render()
}

func rowCells(r Row) []string {
	slotS := ""
	if r.HasSlot {
		slotS = fmt.Sprintf("%d", r.Slot)
	}

	notifS := ""
	if r.HasNotification {
		notifS = notifStyle.Render("!")
	}

	attachS := ""
	if r.Attached {
		attachS = attachedStyle.Render("●")
	}

	dirtyS := ""
	if r.Dirty {
		dirtyS = dirtyStyle.Render("*")
	}

	tasksS := ""
	if r.HasTask {
		tasksS = fmt.Sprintf("%d/%d", r.TasksCompleted, r.TasksTotal)
	}

	todoS := ""
	if len(r.TodoIDs) > 0 {
		todoS = r.TodoIDs[0]
		if len(r.TodoIDs) > 1 {
			todoS += "+"
		}
	}

	statusText := r.Status
	switch {
	case r.Crashed:
		statusText = "crashed"
	case r.NoWorktree:
		statusText = "no-tree"
	}
	statusS := statusText
	switch {
	case r.Crashed || r.NoWorktree:
		statusS = dimStyle.Render(statusText)
	default:
		if style, ok := statusStyles[r.Status]; ok {
			statusS = style.Render(statusText)
		}
	}

	turnS := ""
	if r.HasTurnElapsed {
		turnS = fmtDuration(r.TurnElapsed)
	}
	actS := ""
	if r.HasLastActivity {
		actS = fmtRelative(r.LastActivity)
	}

	promptS := strings.Join(strings.Fields(r.LastPromptPreview), " ")

	return []string{
		slotS,
		r.Project,
		r.Name,
		notifS,
		attachS,
		dirtyS,
		statusS,
		tasksS,
		todoS,
		r.CurrentTool,
		turnS,
		actS,
		promptS,
	}
}

func footerLine(m Model) string {
	hint := "j/k: nav  J/K: move  enter: jump  1-9: jump  alt+1-9: assign  alt+0: unassign  q: quit"
	ts := m.LastRefresh.Format("15:04:05")
	if ts == "00:00:00" {
		ts = "—"
	}
	return fmt.Sprintf("%s    last refresh: %s", hint, ts)
}

func fmtDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d / time.Minute)
		s := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	return fmt.Sprintf("%dh%02dm", h, m)
}

func fmtRelative(t time.Time) string {
	d := time.Since(t).Truncate(time.Second)
	if d < time.Minute {
		return "<1m ago"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
}
