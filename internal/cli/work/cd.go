package work

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/git"
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
				fmt.Print(loc.Repo.Path)
				return nil
			}

			wtPath, err := resolveWorktreePath(loc.Repo, args[0])
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
	worktrees, err := git.ListWorktrees(loc.Repo.Path)
	if err != nil {
		names, err := allProjectWorktrees()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	wtRoot, err := filepath.EvalSymlinks(loc.Repo.WorktreeRoot())
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

func resolveWorktreePath(repo domain.Repo, name string) (string, error) {
	wt := domain.Worktree{RepoPath: repo.Path, Name: name}
	if _, err := os.Stat(wt.Path()); err == nil {
		return wt.Path(), nil
	}
	project, branch, ok := strings.Cut(name, "/")
	if ok {
		projectsDir, err := domain.ProjectsDir()
		if err != nil {
			return "", err
		}
		wtPath := filepath.Join(projectsDir, project, ".work", "tree", branch)
		if _, err := os.Stat(wtPath); err == nil {
			return wtPath, nil
		}
	}
	return "", fmt.Errorf("worktree not found: %s", name)
}
