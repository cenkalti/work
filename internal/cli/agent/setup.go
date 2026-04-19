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

const (
	pluginName      = "work"
	marketplaceName = "work-dev"
)

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Register this repo as a Claude Code plugin in ~/.claude/settings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginDir, err := findPluginDir()
			if err != nil {
				return err
			}
			if err := registerPlugin(pluginDir); err != nil {
				return fmt.Errorf("registering plugin: %w", err)
			}
			if err := cleanupLegacyState(); err != nil {
				return fmt.Errorf("cleaning legacy state: %w", err)
			}
			return nil
		},
	}
}

// findPluginDir returns the absolute path to the plugin root (the directory
// containing .claude-plugin/plugin.json), searching from CWD upward.
func findPluginDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".claude-plugin", "plugin.json")); err == nil {
			return filepath.EvalSymlinks(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".claude-plugin/plugin.json not found in %s or any parent; run `agent setup` from the plugin repo root", cwd)
		}
		dir = parent
	}
}

func registerPlugin(pluginDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	marketplaces, _ := settings["extraKnownMarketplaces"].(map[string]any)
	if marketplaces == nil {
		marketplaces = map[string]any{}
	}
	marketplaces[marketplaceName] = map[string]any{
		"source": map[string]any{
			"source": "directory",
			"path":   pluginDir,
		},
	}
	settings["extraKnownMarketplaces"] = marketplaces

	enabled, _ := settings["enabledPlugins"].(map[string]any)
	if enabled == nil {
		enabled = map[string]any{}
	}
	enabled[pluginName+"@"+marketplaceName] = true
	settings["enabledPlugins"] = enabled

	if err := writeSettings(settingsPath, settings); err != nil {
		return err
	}
	fmt.Printf("plugin: %s@%s registered from %s\n", pluginName, marketplaceName, pluginDir)
	return nil
}

// cleanupLegacyState removes artifacts of the pre-plugin install: the
// `agent hook` entries in ~/.claude/settings.json hooks, the user-scope MCP
// registrations, and the installed command/agent markdown files. The plugin
// now owns all of these; leaving them in place would fire hooks twice and
// shadow plugin-supplied commands.
func cleanupLegacyState() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err := stripLegacyHooks(filepath.Join(home, ".claude", "settings.json")); err != nil {
		return err
	}
	removeLegacyFiles(filepath.Join(home, ".claude", "commands"), []string{
		"plan.md", "execute.md", "recall.md",
		"work-plan.md", "work-execute.md", // retired names
	})
	removeLegacyFiles(filepath.Join(home, ".claude", "agents"), []string{
		"presentation.md",
	})
	removeLegacyMCPs("task", "harness")
	return nil
}

func stripLegacyHooks(settingsPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}

	changed := false
	for event, raw := range hooks {
		groups := parseHookGroups(raw)
		var kept []map[string]any
		anyDropped := false
		for _, g := range groups {
			var entries []map[string]any
			for _, h := range g.Hooks {
				if isAgentHook(h.Command) {
					anyDropped = true
					continue
				}
				entries = append(entries, map[string]any{
					"type":    h.Type,
					"command": h.Command,
				})
			}
			if len(entries) == 0 {
				continue
			}
			group := map[string]any{"hooks": entries}
			if g.Matcher != "" {
				group["matcher"] = g.Matcher
			}
			kept = append(kept, group)
		}
		if !anyDropped {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}

	if !changed {
		fmt.Println("legacy hooks: none to remove")
		return nil
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}
	if err := writeSettings(settingsPath, settings); err != nil {
		return err
	}
	fmt.Println("legacy hooks: removed")
	return nil
}

func removeLegacyFiles(dir string, names []string) {
	for _, n := range names {
		path := filepath.Join(dir, n)
		if err := os.Remove(path); err == nil {
			fmt.Printf("removed legacy file: %s\n", path)
		}
	}
}

func removeLegacyMCPs(names ...string) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return
	}
	for _, name := range names {
		cmd := exec.Command(claudeBin, "mcp", "remove", "--scope", "user", name)
		if out, err := cmd.CombinedOutput(); err == nil {
			fmt.Printf("removed legacy MCP: %s\n", name)
		} else if !strings.Contains(string(out), "not found") && !strings.Contains(string(out), "No MCP") {
			// Swallow the common "not found" cases; surface anything else.
			fmt.Printf("mcp remove %s: %s\n", name, strings.TrimSpace(string(out)))
		}
	}
}

type legacyHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type legacyHookGroup struct {
	Matcher string            `json:"matcher"`
	Hooks   []legacyHookEntry `json:"hooks"`
}

func parseHookGroups(raw any) []legacyHookGroup {
	var groups []legacyHookGroup
	if data, err := json.Marshal(raw); err == nil {
		_ = json.Unmarshal(data, &groups)
	}
	return groups
}

func isAgentHook(cmd string) bool {
	return cmd == "agent hook" || strings.HasPrefix(cmd, "agent hook ")
}

func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0644)
}
