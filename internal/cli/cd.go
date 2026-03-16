package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

func cdCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "cd [name]",
		Short: "Print the path to a worktree",
		Long: `work cd                 # print project root
work cd <goal>          # print goal worktree path
work cd <goal.task>     # print task worktree path

Use with shell integration (shell/work.zsh) to cd into worktrees.`,
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
	loc := detectLocation(cmd)
	if loc.Type == location.Goal || loc.Type == location.Task {
		return listTaskIDsFiltered(loc.TasksDir(), nil), cobra.ShellCompDirectiveNoFileComp
	}
	root := loc.WorktreeRoot()
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
