package dash

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const tickInterval = 1 * time.Second

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
