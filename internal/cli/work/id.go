package work

import (
	"fmt"

	"github.com/spf13/cobra"
)

func idCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "id",
		Short: "Print the current task ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			if loc.IsRoot() {
				fmt.Println(".")
				return nil
			}
			fmt.Println(loc.Branch)
			return nil
		},
	}
}
