package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func nameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "name",
		Short: "Print the current worktree name",
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			if loc.IsRoot() {
				fmt.Println(".")
				return nil
			}
			fmt.Println(loc.Branch)
			return nil
		},
	}
}
