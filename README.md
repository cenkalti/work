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

3. Set up Claude Code hooks, MCP servers, and slash commands:
```bash
agent setup
```

## image-gen MCP

Stdio MCP server that exposes OpenAI `gpt-image-1` as two tools (`generate_image`, `edit_image`). Installed by `go install ./cmd/...` alongside the other binaries.

Register with Claude Code (user scope so every project sees it):

```bash
claude mcp add --scope user --transport stdio image-gen -- image-gen
```

Environment:

- `OPENAI_API_KEY` — required.
- `IMAGE_GEN_DEFAULT_DIR` — optional. If set, relative `output_path` values are resolved against this directory. If unset, relative paths are rejected.

Security note: the server runs locally as you and writes to any absolute path the tool receives. Treat `output_path` as trusted; do not expose this MCP to untrusted prompts.
