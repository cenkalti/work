package agent

import (
	"fmt"

	"github.com/cenkalti/work/internal/wezterm"
)

// SpawnRunWindow opens a new WezTerm window in worktreePath and runs
// `agent run` inside a login shell. Used by `agent jump` and the dashboard
// when no live agent pane exists for a record.
func SpawnRunWindow(worktreePath string) error {
	if worktreePath == "" {
		return fmt.Errorf("agent.SpawnRunWindow: empty worktree path")
	}
	return wezterm.SpawnNewWindow(worktreePath, "/bin/zsh", "-lic", "agent run")
}
