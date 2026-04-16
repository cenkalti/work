package work

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
		Short:             "Change directory to a worktree (requires shell integration)",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: worktreeCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}

			if len(args) == 0 {
				fmt.Print(loc.RootRepo)
				return nil
			}

			wtPath, err := resolveWorktreePath(loc, args[0])
			if err != nil {
				return err
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
	loc, err := detectLocation(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	worktrees, err := git.ListWorktrees(loc.RootRepo)
	if err != nil {
		names, err := allProjectWorktrees()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	wtRoot, err := filepath.EvalSymlinks(loc.WorktreeRoot())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	prefix := wtRoot
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

func resolveWorktreePath(loc *location.Location, name string) (string, error) {
	wtPath := loc.WorktreePath(name)
	if _, err := os.Stat(wtPath); err == nil {
		return wtPath, nil
	}
	project, branch, ok := strings.Cut(name, "/")
	if ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		wtPath = filepath.Join(home, "projects", project, ".work", "tree", branch)
		if _, err := os.Stat(wtPath); err == nil {
			return wtPath, nil
		}
	}
	return "", fmt.Errorf("worktree not found: %s", name)
}
