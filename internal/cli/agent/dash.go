package agent

import (
	"github.com/cenkalti/work/internal/cli/agent/dash"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func dashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dash",
		Short: "Launch the agent dashboard TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(dash.NewModel(), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}
