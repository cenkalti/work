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

## image-gen MCP

Stdio MCP server that exposes OpenAI `gpt-image-1` as two tools (`generate_image`, `edit_image`). Installed by `go install ./cmd/...` alongside the other binaries and registered by the plugin's `.mcp.json`.

Environment:

- `OPENAI_API_KEY` — required.
- `IMAGE_GEN_DEFAULT_DIR` — optional. If set, relative `output_path` values are resolved against this directory. If unset, relative paths are rejected.

Security note: the server runs locally as you and writes to any absolute path the tool receives. Treat `output_path` as trusted; do not expose this MCP to untrusted prompts.
