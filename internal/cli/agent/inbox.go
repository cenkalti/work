package agent

import (
	"fmt"
	"time"

	"github.com/cenkalti/work/internal/inbox"
	"github.com/spf13/cobra"
)

func inboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inbox",
		Short: "Show pending agent notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			msgs, err := inbox.List()
			if err != nil {
				return err
			}
			for _, msg := range msgs {
				age := time.Since(msg.Timestamp).Truncate(time.Second)
				fmt.Printf("%-40s %s ago\n", msg.Name(), age)
			}
			return nil
		},
	}
}
