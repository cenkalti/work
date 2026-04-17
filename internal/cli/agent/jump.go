package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/work/internal/paths"
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
				return fmt.Errorf("no claude pane found for %s", args[0])
			}
			return activateWezTerm(tabID, paneID, tty)
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

type wezPane struct {
	TabID   int    `json:"tab_id"`
	PaneID  int    `json:"pane_id"`
	CWD     string `json:"cwd"`
	TTYName string `json:"tty_name"`
}

func wezTermPath() string {
	if p, err := exec.LookPath("wezterm"); err == nil {
		return p
	}
	for _, p := range []string{"/opt/homebrew/bin/wezterm", "/usr/local/bin/wezterm", "/Applications/WezTerm.app/Contents/MacOS/wezterm"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "wezterm"
}

func findClaudePane(path string) (int, int, string, error) {
	out, err := exec.Command(wezTermPath(), "cli", "list", "--format", "json").Output()
	if err != nil {
		return -1, -1, "", fmt.Errorf("wezterm cli list: %w", err)
	}
	var panes []wezPane
	if err := json.Unmarshal(out, &panes); err != nil {
		return -1, -1, "", fmt.Errorf("parsing wezterm output: %w", err)
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

func activateWezTerm(tabID, paneID int, tty string) error {
	if err := exec.Command(wezTermPath(), "cli", "activate-tab", "--tab-id", fmt.Sprint(tabID)).Run(); err != nil {
		return fmt.Errorf("activate-tab: %w", err)
	}
	if err := exec.Command(wezTermPath(), "cli", "activate-pane", "--pane-id", fmt.Sprint(paneID)).Run(); err != nil {
		return fmt.Errorf("activate-pane: %w", err)
	}
	if tty != "" {
		f, err := os.OpenFile(tty, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("open %s: %w", tty, err)
		}
		val := base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
		_, err = fmt.Fprintf(f, "\x1b]1337;SetUserVar=agent_jump=%s\x07", val)
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	if self := os.Getenv("WEZTERM_PANE"); self != "" && self != fmt.Sprint(paneID) {
		_ = exec.Command(wezTermPath(), "cli", "kill-pane", "--pane-id", self).Run()
	}
	return nil
}
