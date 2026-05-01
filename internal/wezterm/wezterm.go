// Package wezterm wraps the WezTerm CLI: locating the binary, listing panes,
// and activating a pane (with cross-window focus via the agent_jump user var).
//
// The window-raise mechanism: after `wezterm cli activate-pane`, we write an
// OSC 1337 SetUserVar(agent_jump=<unique>) escape to the target pane's TTY.
// The user's wezterm.lua `user-var-changed` handler listens for that name
// and calls `window:focus()`.
package wezterm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Path returns the absolute path to the wezterm binary. WezTerm GUI on macOS
// launches with a minimal PATH, so we fall back through common install
// locations.
func Path() string {
	if p, err := exec.LookPath("wezterm"); err == nil {
		return p
	}
	for _, p := range []string{
		"/opt/homebrew/bin/wezterm",
		"/usr/local/bin/wezterm",
		"/Applications/WezTerm.app/Contents/MacOS/wezterm",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "wezterm"
}

// Pane is a subset of `wezterm cli list --format json` output.
type Pane struct {
	PaneID   int    `json:"pane_id"`
	TabID    int    `json:"tab_id"`
	WindowID int    `json:"window_id"`
	CWD      string `json:"cwd"`
	Title    string `json:"title"`
	TTYName  string `json:"tty_name"`
}

// ListPanes returns every WezTerm pane.
func ListPanes() ([]Pane, error) {
	out, err := exec.Command(Path(), "cli", "list", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("wezterm cli list: %w", err)
	}
	var panes []Pane
	if err := json.Unmarshal(out, &panes); err != nil {
		return nil, fmt.Errorf("parsing wezterm output: %w", err)
	}
	return panes, nil
}

// FindPaneByID returns the pane with the given id, or false if not present.
func FindPaneByID(id int) (Pane, bool) {
	panes, err := ListPanes()
	if err != nil {
		return Pane{}, false
	}
	for _, p := range panes {
		if p.PaneID == id {
			return p, true
		}
	}
	return Pane{}, false
}

// ActivatePane activates the given pane (and its tab) and raises the GUI
// window via the agent_jump user-var trick. Returns an error only on
// activate-pane failures; OSC write failures are silently ignored (TTY may
// have closed, pane may be gone).
func ActivatePane(paneID int) error {
	if err := exec.Command(Path(), "cli", "activate-pane", "--pane-id", strconv.Itoa(paneID)).Run(); err != nil {
		return fmt.Errorf("activate-pane: %w", err)
	}
	if p, ok := FindPaneByID(paneID); ok && p.TTYName != "" {
		writeAgentJump(p.TTYName)
	}
	return nil
}

// ActivatePaneString is a convenience wrapper that accepts a string id.
// Returns an error if id is not a valid integer or activation fails.
func ActivatePaneString(paneID string) error {
	id, err := strconv.Atoi(paneID)
	if err != nil {
		return fmt.Errorf("invalid pane id %q: %w", paneID, err)
	}
	return ActivatePane(id)
}

// ActivateTab activates the tab with the given id.
func ActivateTab(tabID int) error {
	if err := exec.Command(Path(), "cli", "activate-tab", "--tab-id", strconv.Itoa(tabID)).Run(); err != nil {
		return fmt.Errorf("activate-tab: %w", err)
	}
	return nil
}

// SpawnNewWindow spawns a new WezTerm window with cwd and the given command.
// Empty cwd means "use the default". Returns the new pane id.
func SpawnNewWindow(cwd string, args ...string) (int, error) {
	a := []string{"cli", "spawn", "--new-window"}
	if cwd != "" {
		a = append(a, "--cwd", cwd)
	}
	a = append(a, "--")
	a = append(a, args...)
	out, err := exec.Command(Path(), a...).Output()
	if err != nil {
		return 0, fmt.Errorf("spawn: %w", err)
	}
	id, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("spawn: parse pane id: %w", err)
	}
	return id, nil
}

// MaximizePane writes the agent_maximize OSC to the pane's TTY, triggering
// the work.lua user-var-changed handler that calls gw:maximize().
func MaximizePane(paneID int) {
	if p, ok := FindPaneByID(paneID); ok && p.TTYName != "" {
		writeAgentMaximize(p.TTYName)
	}
}

// WriteAgentJump writes the OSC SetUserVar(agent_jump=...) escape to the
// given /dev/tty path. Public so callers that already have a TTY can trigger
// a window-raise without going through ActivatePane.
func WriteAgentJump(tty string) {
	writeAgentJump(tty)
}

func writeAgentJump(tty string) {
	writeUserVar(tty, "agent_jump")
}

func writeAgentMaximize(tty string) {
	writeUserVar(tty, "agent_maximize")
}

func writeUserVar(tty, name string) {
	if !strings.HasPrefix(tty, "/dev/") {
		return
	}
	f, err := os.OpenFile(tty, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	val := base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	_, _ = fmt.Fprintf(f, "\x1b]1337;SetUserVar=%s=%s\x07", name, val)
}
