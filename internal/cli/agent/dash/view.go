package dash

import (
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	cursorStyle   = lipgloss.NewStyle().Reverse(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	notifStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	attachedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dirtyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	statusStyles = map[string]lipgloss.Style{
		agent.StatusRunning:       lipgloss.NewStyle().Foreground(lipgloss.Color("14")), // cyan
		agent.StatusToolRunning:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")), // yellow
		agent.StatusAwaitingInput: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),
		agent.StatusIdle:          lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // green
		agent.StatusStopped:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}

	footerStyle = lipgloss.NewStyle().Faint(true)
)

// Column widths. The last column (prompt) takes the remainder.
const (
	colSlot   = 1
	colNotif  = 1
	colAttach = 1
	colDirty  = 1
	colTasks  = 5
	colTodo   = 7
	colStatus = 14
	colProj   = 12
	colName   = 20
	colTool   = 14
	colTurn   = 7
	colAct    = 8
)

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render(headerLine()))
	b.WriteByte('\n')

	if len(m.Rows) == 0 {
		b.WriteString(dimStyle.Render("(no agents — run `agent run` in a worktree to register one)"))
		b.WriteByte('\n')
	}

	for i, r := range m.Rows {
		b.WriteString(renderRow(r, m.Width, i == m.Cursor))
		b.WriteByte('\n')
	}

	// pad with blank lines so footer hugs the bottom of the alt-screen.
	if m.Height > 0 {
		used := 1 + len(m.Rows) // header + rows
		if len(m.Rows) == 0 {
			used = 2
		}
		footerHeight := 2
		blank := m.Height - used - footerHeight
		for range blank {
			b.WriteByte('\n')
		}
	}

	b.WriteString(footerStyle.Render(footerLine(m)))
	b.WriteByte('\n')
	return b.String()
}

func headerLine() string {
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %s",
		colSlot, "N",
		colProj, "PROJECT",
		colName, "NAME",
		colNotif, "!",
		colAttach, "C",
		colDirty, "D",
		colStatus, "STATUS",
		colTasks, "TASKS",
		colTodo, "TODO",
		colTool, "TOOL",
		colTurn, "TURN",
		colAct, "LAST",
		"PROMPT",
	)
}

func renderRow(r Row, width int, selected bool) string {
	slotS := ""
	if r.HasSlot {
		slotS = fmt.Sprintf("%d", r.Slot)
	}

	notifS := " "
	if r.HasNotification {
		notifS = notifStyle.Render("!")
	}

	attachS := " "
	if r.Attached {
		attachS = attachedStyle.Render("●")
	}

	dirtyS := " "
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
	if r.Crashed {
		statusText = "crashed"
	} else if r.NoWorktree {
		statusText = "no-tree"
	}
	statusS := truncate(statusText, colStatus)
	if style, ok := statusStyles[r.Status]; ok && !r.Crashed && !r.NoWorktree {
		statusS = style.Render(padRight(statusS, colStatus))
	} else if r.Crashed || r.NoWorktree {
		statusS = dimStyle.Render(padRight(statusS, colStatus))
	} else {
		statusS = padRight(statusS, colStatus)
	}

	turnS := ""
	if r.HasTurnElapsed {
		turnS = fmtDuration(r.TurnElapsed)
	}
	actS := ""
	if r.HasLastActivity {
		actS = fmtRelative(r.LastActivity)
	}

	// Compute remaining width for prompt.
	usedWidth := colNotif + 1 + colAttach + 1 + colDirty + 1 + colStatus + 1 + colSlot + 1 + colProj + 1 + colName + 1 + colTasks + 1 + colTodo + 1 + colTool + 1 + colTurn + 1 + colAct + 1
	promptW := max(width-usedWidth, 8)
	promptS := truncate(strings.Join(strings.Fields(r.LastPromptPreview), " "), promptW)

	highlight := fmt.Sprintf("%-*s %-*s %-*s",
		colSlot, slotS,
		colProj, truncate(r.Project, colProj),
		colName, truncate(r.Name, colName),
	)
	if selected {
		highlight = cursorStyle.Render(highlight)
	}

	return fmt.Sprintf("%s %s %s %s %s %-*s %-*s %-*s %-*s %-*s %s",
		highlight,
		notifS,
		attachS,
		dirtyS,
		statusS,
		colTasks, tasksS,
		colTodo, todoS,
		colTool, truncate(r.CurrentTool, colTool),
		colTurn, turnS,
		colAct, actS,
		promptS,
	)
}

func footerLine(m Model) string {
	hint := "j/k: nav  J/K: move  enter: jump  1-9: jump  alt+1-9: assign  alt+0: unassign  q: quit"
	ts := m.LastRefresh.Format("15:04:05")
	if ts == "00:00:00" {
		ts = "—"
	}
	return fmt.Sprintf("%s    last refresh: %s", hint, ts)
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func padRight(s string, n int) string {
	if len([]rune(s)) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len([]rune(s)))
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
