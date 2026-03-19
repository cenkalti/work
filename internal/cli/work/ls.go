package work

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			worktrees, err := git.ListWorktrees(loc.RootRepo)
			if err != nil {
				return err
			}
			// EvalSymlinks to match git's resolved paths (e.g. /tmp -> /private/tmp on macOS).
			wtRoot, err := filepath.EvalSymlinks(loc.WorktreeRoot())
			if err != nil {
				return nil // .work/tree/ doesn't exist yet; no worktrees
			}
			prefix := wtRoot
			if !strings.HasSuffix(prefix, string(filepath.Separator)) {
				prefix += string(filepath.Separator)
			}
			for _, wt := range worktrees {
				if name, ok := strings.CutPrefix(wt, prefix); ok {
					fmt.Println(name)
				}
			}
			return nil
		},
	}
}
