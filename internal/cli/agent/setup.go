package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// hook entry in settings.json
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Async   bool   `json:"async,omitempty"`
}

// hook group is a matcher + list of hooks
type hookGroup struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// desired hooks to ensure exist in settings.json
var desiredHooks = map[string][]hookGroup{
	"SessionStart": {
		{Matcher: "", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook start"},
			{Type: "command", Command: "agent hook context"},
		}},
	},
	"SessionEnd": {
		{Matcher: "", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook end"},
		}},
	},
	"PreToolUse": {
		{Matcher: "", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook status running"},
		}},
		{Matcher: "Bash", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook bash-check"},
		}},
	},
	"UserPromptSubmit": {
		{Matcher: "", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook status running"},
		}},
	},
	"Stop": {
		{Matcher: "", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook status idle"},
		}},
	},
	"Notification": {
		{Matcher: "", Hooks: []hookEntry{
			{Type: "command", Command: "agent hook status idle"},
			{Type: "command", Command: "agent hook notify"},
		}},
	},
}

// MCP servers to register
var desiredMCPs = []struct {
	Name    string
	Command string
	Args    []string
}{
	{Name: "task", Command: "task", Args: []string{"mcp"}},
	{Name: "harness", Command: "harness", Args: nil},
}

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Set up Claude Code hooks, MCP servers, and slash commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupHooks(); err != nil {
				return fmt.Errorf("hooks: %w", err)
			}
			if err := setupMCPs(); err != nil {
				return fmt.Errorf("mcp: %w", err)
			}
			if err := setupCommands(); err != nil {
				return fmt.Errorf("commands: %w", err)
			}
			return nil
		},
	}
}

func setupHooks() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// Read existing settings or start fresh.
	var settings map[string]any
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		settings = make(map[string]any)
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	}

	// Get or create hooks map.
	var hooks map[string]any
	if h, ok := settings["hooks"]; ok {
		hooks, _ = h.(map[string]any)
	}
	if hooks == nil {
		hooks = make(map[string]any)
	}

	changed := false
	for event, groups := range desiredHooks {
		for _, desired := range groups {
			if ensureHookGroup(hooks, event, desired) {
				changed = true
			}
		}
	}

	if !changed {
		fmt.Println("hooks: up to date")
		return nil
	}

	settings["hooks"] = hooks
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return err
	}
	fmt.Println("hooks: updated")
	return nil
}

// ensureHookGroup ensures a hook group with the given matcher exists in the event
// and contains all desired hook entries. Returns true if anything was added.
func ensureHookGroup(hooks map[string]any, event string, desired hookGroup) bool {
	// Parse existing groups for this event.
	var groups []hookGroup
	if raw, ok := hooks[event]; ok {
		if data, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(data, &groups)
		}
	}

	// Find existing group with matching matcher.
	idx := -1
	for i, g := range groups {
		if g.Matcher == desired.Matcher {
			idx = i
			break
		}
	}

	if idx == -1 {
		// No group with this matcher — add the whole thing.
		groups = append(groups, desired)
		hooks[event] = groups
		return true
	}

	// Group exists — ensure each desired hook entry is present.
	changed := false
	for _, dh := range desired.Hooks {
		found := false
		for _, eh := range groups[idx].Hooks {
			if eh.Command == dh.Command {
				found = true
				break
			}
		}
		if !found {
			groups[idx].Hooks = append(groups[idx].Hooks, dh)
			changed = true
		}
	}

	if changed {
		hooks[event] = groups
	}
	return changed
}

func setupMCPs() error {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH")
	}

	for _, mcp := range desiredMCPs {
		args := []string{"mcp", "add", "--transport", "stdio", "--scope", "user", mcp.Name, "--"}
		args = append(args, mcp.Command)
		args = append(args, mcp.Args...)
		cmd := exec.Command(claudeBin, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// "already exists" is fine — idempotent.
			if len(out) > 0 && strings.Contains(string(out), "already exists") {
				fmt.Printf("mcp: %s up to date\n", mcp.Name)
				continue
			}
			return fmt.Errorf("adding MCP %s: %s", mcp.Name, string(out))
		}
		fmt.Printf("mcp: %s registered\n", mcp.Name)
	}
	return nil
}

func setupCommands() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	commandsDir := filepath.Join(home, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return err
	}

	// Find the source commands directory relative to this binary.
	binPath, err := exec.LookPath("agent")
	if err != nil {
		return fmt.Errorf("agent not found in PATH")
	}
	binPath, err = filepath.EvalSymlinks(binPath)
	if err != nil {
		return err
	}

	// Walk up from the binary to find the project root with commands/.
	// The binary is at $GOPATH/bin/agent, so we need another approach.
	// Use the working directory instead — setup should be run from the project root.
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	srcDir := filepath.Join(cwd, "commands")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("commands/ not found in current directory")
	}

	changed := false
	for _, e := range entries {
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(commandsDir, e.Name())

		// Check if symlink already points to the right place.
		if target, err := os.Readlink(dst); err == nil {
			if target == src {
				continue
			}
		}
		_ = os.Remove(dst)
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("symlinking %s: %w", e.Name(), err)
		}
		changed = true
	}

	if changed {
		fmt.Println("commands: updated")
	} else {
		fmt.Println("commands: up to date")
	}
	return nil
}
