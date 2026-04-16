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
