package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/agents"
	"github.com/cenkalti/work/commands"
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
var hookEvents = []string{
	"SessionStart",
	"SessionEnd",
	"PreToolUse",
	"UserPromptSubmit",
	"Stop",
	"StopFailure",
	"Notification",
	"PermissionRequest",
	"Elicitation",
}

func desiredHookGroups() []hookGroup {
	return []hookGroup{{Matcher: "", Hooks: []hookEntry{{Type: "command", Command: "agent hook"}}}}
}

// per-event hook groups that live outside the fan-out list in hookEvents.
type eventHook struct {
	Event string
	Group hookGroup
}

func desiredEventHooks() []eventHook {
	return []eventHook{
		{
			Event: "PostToolUse",
			Group: hookGroup{
				Matcher: "Write|Edit",
				Hooks:   []hookEntry{{Type: "command", Command: "agent hook validate-html"}},
			},
		},
	}
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
			if err := setupAgents(); err != nil {
				return fmt.Errorf("agents: %w", err)
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

	before, _ := json.Marshal(hooks)

	// Strip all existing "agent hook" entries from every event/matcher group.
	// `agent setup` owns this namespace; non-agent entries are preserved.
	for event, raw := range hooks {
		groups := parseHookGroups(raw)
		var kept []hookGroup
		for _, g := range groups {
			var entries []hookEntry
			for _, h := range g.Hooks {
				if !isAgentHook(h.Command) {
					entries = append(entries, h)
				}
			}
			if len(entries) > 0 {
				kept = append(kept, hookGroup{Matcher: g.Matcher, Hooks: entries})
			}
		}
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}

	// Add desired hooks.
	for _, event := range hookEvents {
		for _, desired := range desiredHookGroups() {
			addHookGroup(hooks, event, desired)
		}
	}
	for _, eh := range desiredEventHooks() {
		addHookGroup(hooks, eh.Event, eh.Group)
	}

	after, _ := json.Marshal(hooks)
	if string(before) == string(after) {
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

func isAgentHook(cmd string) bool {
	return cmd == "agent hook" || strings.HasPrefix(cmd, "agent hook ")
}

func parseHookGroups(raw any) []hookGroup {
	var groups []hookGroup
	if data, err := json.Marshal(raw); err == nil {
		_ = json.Unmarshal(data, &groups)
	}
	return groups
}

// addHookGroup appends desired entries to the matching event+matcher group,
// creating the group if it doesn't exist.
func addHookGroup(hooks map[string]any, event string, desired hookGroup) {
	groups := parseHookGroups(hooks[event])
	for i, g := range groups {
		if g.Matcher == desired.Matcher {
			groups[i].Hooks = append(groups[i].Hooks, desired.Hooks...)
			hooks[event] = groups
			return
		}
	}
	groups = append(groups, desired)
	hooks[event] = groups
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

	entries, err := commands.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("reading embedded commands: %w", err)
	}

	changed := false
	for _, e := range entries {
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := commands.FS.ReadFile(e.Name())
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", e.Name(), err)
		}
		dst := filepath.Join(commandsDir, e.Name())

		if existing, err := os.ReadFile(dst); err == nil && bytes.Equal(existing, data) {
			continue
		}
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
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

func setupAgents() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	agentsDir := filepath.Join(home, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	entries, err := agents.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("reading embedded agents: %w", err)
	}

	changed := false
	for _, e := range entries {
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := agents.FS.ReadFile(e.Name())
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", e.Name(), err)
		}
		dst := filepath.Join(agentsDir, e.Name())

		if existing, err := os.ReadFile(dst); err == nil && bytes.Equal(existing, data) {
			continue
		}
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
		}
		changed = true
	}

	if changed {
		fmt.Println("agents: updated")
	} else {
		fmt.Println("agents: up to date")
	}
	return nil
}
