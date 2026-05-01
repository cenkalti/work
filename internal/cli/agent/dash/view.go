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
	accent  = lipgloss.Color("#7D56F4")
	accent2 = lipgloss.Color("#43BF6D")
	muted   = lipgloss.Color("#5C5C5C")
	cursor  = lipgloss.Color("#3A3A55")
	soft    = lipgloss.Color("#A29BFE")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accent).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(muted).
			MarginLeft(1)

	borderStyle = lipgloss.NewStyle().Foreground(muted)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(soft).
			PaddingRight(1)

	cellStyle = lipgloss.NewStyle().PaddingRight(1)

	cursorStyle = lipgloss.NewStyle().
			Background(cursor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			PaddingRight(1)

	dimStyle      = lipgloss.NewStyle().Foreground(muted)
	notifStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5F5F"))
	attachedStyle = lipgloss.NewStyle().Foreground(accent2)
	dirtyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD866"))

	statusStyles = map[string]lipgloss.Style{
		agent.StatusRunning:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5FD7FF")),
		agent.StatusToolRunning:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD866")),
		agent.StatusAwaitingInput: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5F87")),
		agent.StatusIdle:          lipgloss.NewStyle().Foreground(accent2),
		agent.StatusStopped:       lipgloss.NewStyle().Faint(true).Foreground(muted),
	}

	footerStyle = lipgloss.NewStyle().Faint(true).Foreground(muted).MarginTop(1)
	keyStyle    = lipgloss.NewStyle().Bold(true).Foreground(soft)
	emptyStyle  = lipgloss.NewStyle().Foreground(muted).Italic(true).Padding(0, 2)
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
	"#", "PROJECT", "NAME", "!", "C", "D",
	"STATUS", "TASKS", "TODO", "TOOL", "TURN", "LAST", "PROMPT",
}

func (m Model) View() tea.View {
	header := titleStyle.Render("◆ AGENTS")
	if !m.LastRefresh.IsZero() {
		header = lipgloss.JoinHorizontal(
			lipgloss.Top,
			header,
			subtitleStyle.Render(fmt.Sprintf("· %d agents · refreshed %s", len(m.Rows), m.LastRefresh.Format("15:04:05"))),
		)
	}

	body := m.renderTable()
	if len(m.Rows) == 0 {
		body = emptyStyle.Render("no agents — run `agent run` in a worktree to register one")
	}

	footer := footerStyle.Render(footerLine())

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	if m.Height > 0 {
		used := lipgloss.Height(content)
		blank := m.Height - used
		if blank > 0 {
			content += strings.Repeat("\n", blank)
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderTable() string {
	rows := make([][]string, 0, len(m.Rows))
	for _, r := range m.Rows {
		rows = append(rows, rowCells(r))
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(true).
		Headers(headers...).
		Rows(rows...).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if row == m.Cursor {
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
		notifS = notifStyle.Render("●")
	}

	attachS := ""
	if r.Attached {
		attachS = attachedStyle.Render("●")
	}

	dirtyS := ""
	if r.Dirty {
		dirtyS = dirtyStyle.Render("◆")
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

func footerLine() string {
	parts := []string{
		keyStyle.Render("j/k") + " nav",
		keyStyle.Render("J/K") + " move",
		keyStyle.Render("⏎") + " jump",
		keyStyle.Render("1-9") + " slot",
		keyStyle.Render("⌥1-9") + " assign",
		keyStyle.Render("⌥0") + " unassign",
		keyStyle.Render("q") + " quit",
	}
	return strings.Join(parts, "  ·  ")
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
