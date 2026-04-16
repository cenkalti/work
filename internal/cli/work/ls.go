package work

import (
	"fmt"
	"os"
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
				return listAllProjects()
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
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	projectsDir := filepath.Join(home, "projects")
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
