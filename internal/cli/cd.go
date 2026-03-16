package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func cdCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "cd <name>",
		Short:             "Print the path to a worktree (use with shell integration)",
		Long:              `Print the worktree path for a goal or task. Use the shell function from shell/work.zsh to cd into it.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			loc := detectLocation(cmd)

			wtPath := loc.WorktreePath(name)
			if _, err := os.Stat(wtPath); err != nil {
				return fmt.Errorf("worktree not found: %s", wtPath)
			}

			fmt.Print(wtPath)
			return nil
		},
	}
}

func worktreeCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	root := detectLocation(cmd).WorktreeRoot()
	return listWorktreeNames(root), cobra.ShellCompDirectiveNoFileComp
}

func listWorktreeNames(workTreeRoot string) []string {
	prefix := workTreeRoot
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	var names []string
	for _, path := range git.ListWorktrees(workTreeRoot) {
		if name, ok := strings.CutPrefix(path, prefix); ok {
			names = append(names, name)
		}
	}
	return names
}
