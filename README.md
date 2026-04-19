# Work

Multi-task orchestrator for Claude Code. Decomposes plans into tasks with dependencies, then runs each task as a separate Claude Code instance in its own git worktree.

## Installation

1. Build and install the binaries:
```bash
go install ./cmd/...
```

2. Add shell integration to `~/.zshrc`:
```bash
source /Users/cenk/projects/work/shell/work.zsh
```

3. Register this repo as a Claude Code plugin (from the repo root):
```bash
agent setup
```
This writes `work@work-dev` to `~/.claude/settings.json` (`extraKnownMarketplaces` + `enabledPlugins`) and removes any state written by pre-plugin installs (stale hooks, user-scope MCP registrations, and copied command/agent files). Claude Code loads the plugin automatically on every launch.
