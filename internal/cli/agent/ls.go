package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	var flagRunning bool
	var flagActive bool

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List agents across all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			var sessionIDs map[string]struct{}
			if flagRunning || flagActive {
				sessionIDs = agent.RunningSessionIDs()
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			projectsDir := filepath.Join(home, "projects")
			entries, err := os.ReadDir(projectsDir)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				projectPath := filepath.Join(projectsDir, entry.Name())
				worktrees, err := git.ListWorktrees(projectPath)
				if err != nil {
					continue
				}
				for _, wt := range worktrees {
					state, err := agent.Read(wt)
					if err != nil {
						continue
					}

					if flagRunning || flagActive {
						if _, ok := sessionIDs[strings.ToLower(state.ID)]; !ok {
							continue
						}
					}
					if flagActive && state.Status != agent.StatusRunning {
						continue
					}

					name := nameForWorktree(entry.Name(), projectPath, wt)
					if name == "" {
						continue
					}
					fmt.Println(name)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagRunning, "running", "r", false, "only show agents with a running claude session")
	cmd.Flags().BoolVarP(&flagActive, "active", "a", false, "only show agents actively working (not idle)")

	return cmd
}

func nameForWorktree(project, projectPath, wt string) string {
	if wt == projectPath {
		return project
	}
	wtRoot := filepath.Join(projectPath, ".work", "tree")
	wtRootResolved, err := filepath.EvalSymlinks(wtRoot)
	if err != nil {
		return ""
	}
	prefix := wtRootResolved + string(filepath.Separator)
	if name, ok := strings.CutPrefix(wt, prefix); ok {
		return project + "/" + name
	}
	return ""
}
