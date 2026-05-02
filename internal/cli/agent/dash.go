package agent

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/work/internal/cli/agent/dash"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func dashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dash",
		Short: "Launch the agent dashboard TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			markDashPane()
			if path := writeDashPaneFile(); path != "" {
				defer os.Remove(path)
			}
			p := tea.NewProgram(dash.NewModel())
			_, err := p.Run()
			return err
		},
	}
}

// writeDashPaneFile records the WezTerm pane id of the dashboard at
// ~/.work/dash.pane so external callers (e.g. the menu bar) can locate it
// without needing access to WezTerm user vars. Returns the path written, or
// "" if nothing was written.
func writeDashPaneFile() string {
	pane := os.Getenv("WEZTERM_PANE")
	if pane == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".work", "dash.pane")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(path, []byte(pane+"\n"), 0o644); err != nil {
		return ""
	}
	return path
}

// markDashPane sets the WezTerm user var agent_role=dash on this pane so the
// toggle handler in work.lua can find the dashboard regardless of who spawned
// it (direct invocation, post-config-reload, etc.).
func markDashPane() {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	val := base64.StdEncoding.EncodeToString([]byte("dash"))
	fmt.Fprintf(f, "\x1b]1337;SetUserVar=agent_role=%s\x07", val)
}
