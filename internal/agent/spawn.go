package agent

import (
	"fmt"

	"github.com/cenkalti/work/internal/wezterm"
)

// SpawnRunWindow opens a new WezTerm window in worktreePath and runs
// `agent run` inside a login shell. Used by `agent jump` and the dashboard
// when no live agent pane exists for a record. Returns the new pane id.
func SpawnRunWindow(worktreePath string) (int, error) {
	if worktreePath == "" {
		return 0, fmt.Errorf("agent.SpawnRunWindow: empty worktree path")
	}
	return wezterm.SpawnNewWindow(worktreePath, "/bin/zsh", "-lic", "agent run")
}
