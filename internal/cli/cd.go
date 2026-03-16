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
		Use:               "cd [name]",
		Short:             "Print the path to a worktree (use with shell integration)",
		Long:              `Print the worktree path for a goal or task. With no arguments, prints the project root. Use the shell function from shell/work.zsh to cd into it.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)

			if len(args) == 0 {
				fmt.Print(loc.RootRepo)
				return nil
			}

			wtPath := loc.WorktreePath(args[0])
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
	worktrees, err := git.ListWorktrees(root)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	var names []string
	for _, path := range worktrees {
		if name, ok := strings.CutPrefix(path, prefix); ok {
			names = append(names, name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
