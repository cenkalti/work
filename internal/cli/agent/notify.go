package agent

import (
	"path/filepath"
	"time"

	agentpkg "github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/inbox"
	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

func notifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "notify",
		Short:  "Record inbox notification (Notification hook)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := agentpkg.ReadHookInput()
			if err != nil {
				return err
			}
			loc, err := location.Detect()
			if err != nil {
				return err
			}
			return inbox.Write(&inbox.Message{
				Project:   filepath.Base(loc.RootRepo),
				Branch:    loc.Branch,
				SessionID: input.SessionID,
				Timestamp: time.Now(),
			})
		},
	}
}
