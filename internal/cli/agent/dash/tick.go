package dash

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

const tickInterval = 1 * time.Second

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
