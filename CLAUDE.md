# CLAUDE.md

## Building

```bash
go install ./cmd/...
```

## Installation

See [README.md](README.md).

## Commands

### Worktree Commands (absolute dot-separated branch names)

```bash
work run [name]              # no arg → start session here; name → create worktree and start session
work id                      # print the current task's dot-separated ID (see below)
work ls                      # list all worktrees
work mv <src> <dst>          # move/rename task (use . for root)
work rm <name>               # remove worktree and branch
work cd [name]               # change directory to worktree (requires shell integration)
```

### Task Commands (separate `task` binary, operates on ./workspace/tasks/)

```bash
task ls                    # list subtasks (--ready, --active, --blocked, --pending, --completed)
task show <id>               # show task details as YAML
task tree [id]               # dependency tree
task set-status <id> <status> # set task status (pending, active, completed)
task mcp                     # start MCP server for task creation (hidden)
```

### Harness MCP Server (`harness` binary, Karpathy Guidelines)

MCP server that exposes six tools for applying and evaluating the Karpathy coding principles. Requires `ANTHROPIC_API_KEY`.

```bash
harness [-model <id>]        # start MCP server on stdin/stdout
```

Tools exposed:
- `think(task)` — Surface assumptions, interpretations, tradeoffs before coding (Principle 1)
- `simplify(code, context?)` — Score complexity 0-10, list over-engineering issues (Principle 2)
- `surgical(diff?, task?)` — Check if diff is surgical; auto-runs `git diff --staged` if no diff given (Principle 3)
- `goal(task)` — Transform vague task into numbered plan with `→ verify:` steps (Principle 4)
- `score(response)` — Score any LLM response against all four principles
- `bench(guidelines?)` — Run 8 built-in benchmark cases, compare baseline vs. with-guidelines
```

## Naming and Identifiers

All identifiers are plain strings. No UUIDs or auto-generated IDs.

| Identifier | Format | Example |
|---|---|---|
| Task ID | kebab-case slug | `build-login-form` |
| Task file | `<task-id>.json` | `.work/space/user-auth/tasks/build-login-form.json` |
| Root task branch | User-provided, no dots | `user-auth` |
| Child task branch | `<parent-branch>.<task-id>` | `user-auth.build-login-form` |
| Worktree path | `.work/tree/<branch>/` | `.work/tree/user-auth.build-login-form/` |
| Workspace path | `.work/space/<branch>/` | `.work/space/user-auth.build-login-form/` |
| Work log | `.work/space/<branch>/log.md` | `.work/space/user-auth.build-login-form/log.md` |

### `work id` — Task Identity

`work id` prints the fully-qualified ID of the current task. The ID is a **dot-separated path** that encodes the task's position in the hierarchy. At the root repo (no active task), it prints `.`.

The dot-separated structure works like a filesystem path but uses `.` as the delimiter:

```
user-auth                       # root task
user-auth.build-login-form      # subtask of user-auth
user-auth.build-login-form.api  # subtask of build-login-form
```

Each segment is a kebab-case task ID. Reading left to right gives the full ancestry: `a.b.c` means task `c`, which is a child of `b`, which is a child of root task `a`.

This ID is also the git branch name. The dot-separated structure means you can always derive:
- **Parent branch**: everything before the last dot (`user-auth.build-login-form` → `user-auth`)
- **Task ID**: the last segment after the final dot (`user-auth.build-login-form` → `build-login-form`)
- **Workspace path**: `.work/space/<full-id>/`
- **Worktree path**: `.work/tree/<full-id>/`

Examples:

```bash
$ work id
.                                    # at repo root, no active task

$ cd .work/tree/user-auth/
$ work id
user-auth                            # root task

$ cd .work/tree/user-auth.build-login-form/
$ work id
user-auth.build-login-form           # nested subtask
```

Key points:
- Everything is a task. Root tasks have no parent; any task can be decomposed into subtasks.
- Branch name encodes the full ancestry via dots: `a.b.c` means task `c` under `b` under `a`.
- Dependencies are between siblings only (same parent).
- **Single `.work/` directory** in the root repo. All worktrees symlink to it.
- Task worktrees are created in the **root repo** (resolved via `git rev-parse --git-common-dir`).
- `.work/` in all worktrees is a **symlink** back to the root repo's `.work/`, so all agents share the full workspace.
- Subtasks are stored in the parent's workspace: `.work/space/<parent-branch>/tasks/<id>.json`.

## Directory Structure

```
.work/
├── space/
│   ├── <branch>/               # task workspace
│   │   ├── plan.md
│   │   ├── tasks/
│   │   │   └── <subtask-id>.json
│   │   └── (scratch files)
│   └── <branch>.<subtask-id>/ # subtask workspace
│       ├── log.md
│       └── (scratch files)
└── tree/                       # git worktrees (excluded from backup)
    ├── <branch>/
    └── <branch>.<subtask-id>/
```

Backup: copy `.work/space/`. The `tree/` directory contains only git worktrees (reproducible from branches).

## Architecture

Work is a multi-task orchestrator for Claude Code. It decomposes plans into tasks with dependencies, then runs each task as a separate Claude Code instance in its own git worktree.

### Key packages

- **`internal/cli/work/`** — Cobra commands for the `work` binary. Worktree management, session lifecycle, context hook.
- **`internal/cli/task/`** — Cobra commands for the `task` binary. Task listing, show, tree, set-status, MCP server.
- **`internal/location/`** — Detects current working context from CWD and git branch. `Branch` is the full dot-separated path; empty at the root repo.
- **`internal/paths/`** — Path construction helpers. `ParentBranch`/`BranchID` split dot-notation branches.
- **`internal/task/`** — Task data model (ID, summary, depends_on, status, files, description, acceptance, context). Tasks are JSON files in parent workspaces.
- **`internal/session/`** — Worktree setup and Claude execution. `Run` creates the worktree, workspace, symlinks, then execs into `claude`.
- **`internal/git/`** — Git worktree creation/removal and branch helpers.
- **`internal/mcp/`** — MCP server implementation exposing `create_task` tool so Claude can create subtasks during planning.

### Task lifecycle

`work run <name>` → creates git worktree + branch → creates workspace → symlinks `workspace/` → execs into `claude`. On session start, the `SessionStart` hook calls `work context`, which injects task details into the conversation.

## Slash Commands

Slash commands live in `commands/` at the repo root. Symlinked to `~/.claude/commands/` during installation.

1. **`commands/work-plan.md`** — `/work-plan`. Human-driven planning session: goal capture, research, plan, task decomposition via the task MCP tool.

2. **`commands/work-execute.md`** — `/work-execute`. Run inside a task's Claude Code session to work on the assigned task and maintain a work log.
