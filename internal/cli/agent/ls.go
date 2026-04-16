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
	var flagAll bool
	var flagIdle bool
	var flagRunning bool

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List agents across all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			var sessionIDs map[string]struct{}
			if !flagAll {
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

					if !flagAll {
						if _, ok := sessionIDs[strings.ToLower(state.ID)]; !ok {
							continue
						}
					}
					if flagRunning && state.Status != agent.StatusRunning {
						continue
					}
					if flagIdle && state.Status != agent.StatusIdle {
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

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "list all agents regardless of status")
	cmd.Flags().BoolVar(&flagRunning, "running", false, "only show agents actively working (not idle)")
	cmd.Flags().BoolVar(&flagIdle, "idle", false, "only show agents with a running but idle session")

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
