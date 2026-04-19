# Execute

You are working on a task defined in your CLAUDE.md. Follow these steps:

## Step 1: Read Your Task

Task details were injected at session start by the `SessionStart` hook. To re-read them, run `task show <your-task-id>` — the id is the last segment of `work id` (e.g., `build-login-form` from `user-auth.build-login-form`).

For background, read the parent workspace's `plan.md`, `context.md`, and `research.md` (if present). The parent's workspace is `.work/space/<parent-branch>/`, where `<parent-branch>` is everything before the last dot in `work id`.

## Step 2: Start the Work Log

Create your work log at `workspace/log.md`. Use this structure:

```markdown
# Work Log

## Status: in_progress

## Task
<copy your task summary here>

## Acceptance
<copy the acceptance criteria here — check them off as you satisfy each>

## Progress

## What's Left

## Blockers

## Notes
```

## Step 3: Do the Work

Implement the task according to the acceptance criteria. Update `log.md` at checkpoints:

- When you satisfy an acceptance criterion — check it off in **Acceptance**, note how in **Progress**
- When you hit a blocker — record it in **Blockers** and stop; surface to the human before deciding to work around it
- When you discover follow-up work that doesn't belong in this task — note it in **Notes** and use `create_task` to file it as a separate task
- When you realize the task scope is wrong — stop, document in **Notes**, and surface to the human before continuing

Use `workspace/` for scratch files and intermediate outputs.

## Step 4: Finish Up

Before marking complete, verify:
- Every acceptance criterion is checked off in **Acceptance**
- Relevant tests, type-checks, and lints pass
- `log.md` has a final summary in **Progress** and **What's Left** is empty or lists follow-ups

Then run `task set-status <your-task-id> completed` (the id is the last segment of `work id`).
