package wezterm

import "fmt"

// SpawnAgentRunWindow opens a new WezTerm window in worktreePath and runs
// `agent run` inside a login shell. Used by `agent jump` and the dashboard
// when no live agent pane exists for a record. Returns the new pane id.
func SpawnAgentRunWindow(worktreePath string) (int, error) {
	if worktreePath == "" {
		return 0, fmt.Errorf("wezterm.SpawnAgentRunWindow: empty worktree path")
	}
	return SpawnNewWindow(worktreePath, "/bin/zsh", "-lic", "agent run")
}
