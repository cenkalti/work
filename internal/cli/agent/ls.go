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

type listOpts struct {
	all     bool
	running bool
	idle    bool
}

func psCmd() *cobra.Command {
	var opts listOpts

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List agents across all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := listAgents(opts)
			if err != nil {
				return err
			}
			for _, name := range names {
				fmt.Println(name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.all, "all", "a", false, "list all agents regardless of status")
	cmd.Flags().BoolVar(&opts.running, "running", false, "only show agents actively working (not idle)")
	cmd.Flags().BoolVar(&opts.idle, "idle", false, "only show agents with a running but idle session")

	return cmd
}

func listAgents(opts listOpts) ([]string, error) {
	var sessionIDs map[string]struct{}
	if !opts.all {
		sessionIDs = agent.RunningSessionIDs()
	}

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
		worktrees, err := git.ListWorktrees(projectPath)
		if err != nil {
			continue
		}
		for _, wt := range worktrees {
			state, err := agent.Read(wt)
			if err != nil {
				continue
			}

			if !opts.all {
				if _, ok := sessionIDs[strings.ToLower(state.ID)]; !ok {
					continue
				}
			}
			if opts.running && state.Status != agent.StatusRunning {
				continue
			}
			if opts.idle && state.Status != agent.StatusIdle {
				continue
			}

			name := nameForWorktree(entry.Name(), projectPath, wt)
			if name == "" {
				continue
			}
			names = append(names, name)
		}
	}
	return names, nil
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
