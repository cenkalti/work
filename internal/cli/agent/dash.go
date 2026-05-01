package agent

import (
	"encoding/base64"
	"fmt"
	"os"

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
			p := tea.NewProgram(dash.NewModel())
			_, err := p.Run()
			return err
		},
	}
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
