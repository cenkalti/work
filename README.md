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

3. Symlink slash commands to `~/.claude/commands/`:
```bash
mkdir -p ~/.claude/commands
ln -sf /Users/cenk/projects/work/commands/*.md ~/.claude/commands/
```

4. Add the `agent hook context` SessionStart hook to `~/.claude/settings.json`:
```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "agent hook context"
          }
        ]
      }
    ]
  }
}
```

5. Add the `agent hook bash-check` PreToolUse hook to `~/.claude/settings.json`:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "agent hook bash-check"
          }
        ]
      }
    ]
  }
}
```

6. Add the `task` MCP server globally:
```bash
claude mcp add --transport stdio --scope user task -- task mcp
```
