package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/git"
	"github.com/spf13/cobra"
)

func jumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jump <id>",
		Short: "Activate the WezTerm tab for an agent",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return agentNames(), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			path, err := resolveAgentPath(id)
			if err != nil {
				return err
			}
			var sessionID string
			if state, err := agent.Read(path); err == nil {
				sessionID = state.ID
			}
			tabID, paneID, tty, err := findWezTermPane(path, sessionID)
			if err != nil {
				return err
			}
			if tabID < 0 {
				return fmt.Errorf("no wezterm tab for %s (%s)", id, path)
			}
			return activateWezTerm(tabID, paneID, tty)
		},
	}
	return cmd
}

func resolveAgentPath(id string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	project, branch, _ := strings.Cut(id, "/")
	projectPath := filepath.Join(home, "projects", project)
	var path string
	if branch == "" {
		path = projectPath
	} else {
		path = filepath.Join(projectPath, ".work", "tree", branch)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("unknown agent: %s", id)
	}
	return resolved, nil
}

type wezPane struct {
	TabID    int    `json:"tab_id"`
	PaneID   int    `json:"pane_id"`
	CWD      string `json:"cwd"`
	TTYName  string `json:"tty_name"`
	IsActive bool   `json:"is_active"`
}

func findWezTermPane(path, sessionID string) (int, int, string, error) {
	out, err := exec.Command("wezterm", "cli", "list", "--format", "json").Output()
	if err != nil {
		return -1, -1, "", fmt.Errorf("wezterm cli list: %w", err)
	}
	var panes []wezPane
	if err := json.Unmarshal(out, &panes); err != nil {
		return -1, -1, "", fmt.Errorf("parsing wezterm output: %w", err)
	}
	if sessionID != "" {
		if tty := claudeTTY(sessionID); tty != "" {
			for _, p := range panes {
				if p.TTYName == tty {
					return p.TabID, p.PaneID, p.TTYName, nil
				}
			}
		}
	}
	target := path + string(filepath.Separator)
	tabID, paneID := -1, -1
	var tty string
	for _, p := range panes {
		cwd := strings.TrimPrefix(p.CWD, "file://")
		resolved, err := filepath.EvalSymlinks(cwd)
		if err != nil {
			resolved = cwd
		}
		if resolved != path && !strings.HasPrefix(resolved+string(filepath.Separator), target) {
			continue
		}
		if p.IsActive || tabID < 0 {
			tabID, paneID, tty = p.TabID, p.PaneID, p.TTYName
			if p.IsActive {
				break
			}
		}
	}
	return tabID, paneID, tty, nil
}

func claudeTTY(sessionID string) string {
	out, err := exec.Command("ps", "-eo", "tty,args").Output()
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		tty := fields[0]
		if tty == "??" || fields[1] != "claude" {
			continue
		}
		for i := 2; i < len(fields)-1; i++ {
			if (fields[i] == "--resume" || fields[i] == "--session-id") && strings.EqualFold(fields[i+1], sessionID) {
				return "/dev/" + tty
			}
		}
	}
	return ""
}

func activateWezTerm(tabID, paneID int, tty string) error {
	if err := exec.Command("wezterm", "cli", "activate-tab", "--tab-id", fmt.Sprint(tabID)).Run(); err != nil {
		return fmt.Errorf("activate-tab: %w", err)
	}
	if err := exec.Command("wezterm", "cli", "activate-pane", "--pane-id", fmt.Sprint(paneID)).Run(); err != nil {
		return fmt.Errorf("activate-pane: %w", err)
	}
	if tty != "" {
		f, err := os.OpenFile(tty, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("open %s: %w", tty, err)
		}
		val := base64.StdEncoding.EncodeToString([]byte("1"))
		_, err = fmt.Fprintf(f, "\x1b]1337;SetUserVar=agent_jump=%s\x07", val)
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	if self := os.Getenv("WEZTERM_PANE"); self != "" && self != fmt.Sprint(paneID) {
		_ = exec.Command("wezterm", "cli", "kill-pane", "--pane-id", self).Run()
	}
	return nil
}

func agentNames() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	projectsDir := filepath.Join(home, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
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
			if _, err := agent.Read(wt); err != nil {
				continue
			}
			name := nameForWorktree(entry.Name(), projectPath, wt)
			if name == "" {
				continue
			}
			names = append(names, name)
		}
	}
	return names
}
