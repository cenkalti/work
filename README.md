# Work

Multi-task orchestrator for Claude Code. Decomposes plans into tasks with dependencies, then runs each task as a separate Claude Code instance in its own git worktree.

## Installation

1. Build and install the binary:
```bash
go build -o ~/go/bin/work ./cmd/work/
```

2. Add shell integration to `~/.zshrc`:
```bash
source /Users/cenk/projects/work/shell/work.zsh
```

3. Symlink slash commands to `~/.claude/commands/`:
```bash
mkdir -p ~/.claude/commands
ln -sf /Users/cenk/projects/work/commands/*.md ~/.claude/commands/
```

4. Add the `work context` SessionStart hook to `~/.claude/settings.json`:
```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "work context"
          }
        ]
      }
    ]
  }
}
```

5. Add the `work` MCP server globally:
```bash
claude mcp add --transport stdio --scope user work -- work mcp
```
