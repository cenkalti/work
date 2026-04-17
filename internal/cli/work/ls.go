package work

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			// Outside any git repo: fall back to enumerating ~/projects/*.
			worktrees, err := git.ListWorktrees(loc.RootRepo)
			if err != nil {
				return listAllProjects()
			}
			// EvalSymlinks to match git's resolved paths (e.g. /tmp -> /private/tmp on macOS).
			// Failure here means the repo has no .work/tree/ yet — it's a git repo but
			// not adopted by work. Stay scoped to this repo (empty output) rather than
			// spilling into other projects' worktrees.
			wtRoot, err := filepath.EvalSymlinks(loc.WorktreeRoot())
			if err != nil {
				return nil
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

func listAllProjects() error {
	names, err := allProjectWorktrees()
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

func allProjectWorktrees() ([]string, error) {
	projectsDir, err := paths.ProjectsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectPath := filepath.Join(projectsDir, entry.Name())
		wtRoot, err := filepath.EvalSymlinks(filepath.Join(projectPath, ".work", "tree"))
		if err != nil {
			continue
		}
		worktrees, err := git.ListWorktrees(projectPath)
		if err != nil {
			continue
		}
		prefix := wtRoot + string(filepath.Separator)
		for _, wt := range worktrees {
			if name, ok := strings.CutPrefix(wt, prefix); ok {
				names = append(names, entry.Name()+"/"+name)
			}
		}
	}
	return names, nil
}
