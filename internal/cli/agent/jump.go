package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	agentpkg "github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/paths"
	"github.com/cenkalti/work/internal/wezterm"
	"github.com/spf13/cobra"
)

func jumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jump <project>[/<branch>]",
		Short: "Activate the WezTerm pane running claude for the given agent",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names, _ := listAgents(listOpts{})
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveAgentPath(args[0])
			if err != nil {
				return err
			}
			tabID, paneID, tty, err := findClaudePane(path)
			if err != nil {
				return err
			}
			if paneID < 0 {
				// No live claude pane in this worktree; spawn a new window
				// running `agent run` there.
				_, err := agentpkg.SpawnRunWindow(path)
				return err
			}
			if err := wezterm.ActivateTab(tabID); err != nil {
				return err
			}
			if err := wezterm.ActivatePane(paneID); err != nil {
				return err
			}
			if tty != "" {
				wezterm.WriteAgentJump(tty)
			}
			if self := os.Getenv("WEZTERM_PANE"); self != "" && self != fmt.Sprint(paneID) {
				_ = exec.Command(wezterm.Path(), "cli", "kill-pane", "--pane-id", self).Run()
			}
			return nil
		},
	}
}

func resolveAgentPath(id string) (string, error) {
	projectsDir, err := paths.ProjectsDir()
	if err != nil {
		return "", err
	}
	project, branch, _ := strings.Cut(id, "/")
	projectPath := filepath.Join(projectsDir, project)
	path := projectPath
	if branch != "" {
		path = filepath.Join(projectPath, ".work", "tree", branch)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("unknown agent: %s", id)
	}
	return resolved, nil
}

// findClaudePane returns the tab/pane/tty of the WezTerm pane running claude
// in the given worktree path. paneID is -1 if no match.
func findClaudePane(path string) (int, int, string, error) {
	panes, err := wezterm.ListPanes()
	if err != nil {
		return -1, -1, "", err
	}
	claudeTTYs := runningClaudeTTYs()
	for _, p := range panes {
		if _, ok := claudeTTYs[p.TTYName]; !ok {
			continue
		}
		cwd := strings.TrimPrefix(p.CWD, "file://")
		resolved, err := filepath.EvalSymlinks(cwd)
		if err != nil {
			resolved = cwd
		}
		if resolved == path {
			return p.TabID, p.PaneID, p.TTYName, nil
		}
	}
	return -1, -1, "", nil
}

func runningClaudeTTYs() map[string]struct{} {
	ttys := map[string]struct{}{}
	out, err := exec.Command("ps", "-eo", "tty,args").Output()
	if err != nil {
		return ttys
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		tty := fields[0]
		if tty == "??" || fields[1] != "claude" {
			continue
		}
		ttys["/dev/"+tty] = struct{}{}
	}
	return ttys
}
