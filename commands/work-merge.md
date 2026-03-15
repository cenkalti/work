# Merge

You are merging a goal branch back into main. Follow these steps.

## Step 1: Check for Uncommitted Changes

```bash
git status --short
```

Commit or discard anything that shouldn't go to main before proceeding. In particular:
- `CLAUDE.md` in the worktree is a generated file injected by `work run` — restore it: `git checkout CLAUDE.md`
- `.mcp.json` is a local config file — do not commit it

## Step 2: Confirm the Branch is Ready

Show what will be merged:

```bash
git log --oneline main..HEAD
```

Make sure all tasks are complete:

```bash
work list
```

If anything is still pending, discuss with the human before proceeding.

## Step 3: Run Tests

```bash
go test ./...
```

Do not merge with failing tests.

## Step 4: Merge into Main

You are in a worktree — `git checkout main` will fail. Merge via the root repo instead:

```bash
git -C $(git rev-parse --git-common-dir | xargs dirname | xargs dirname) merge $(git branch --show-current)
```

Or equivalently, find the root repo from `git rev-parse --git-common-dir` (strip the trailing `/.git`) and run `git merge` there.

## Step 5: Verify

```bash
git -C <root-repo> log --oneline -10
go test ./...
```

Confirm the commits landed on main and all tests still pass.
