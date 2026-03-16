# Plan

You are helping a human build a software project. The human is the decision-maker. You facilitate, research, write documents, and propose — but the human approves every step before moving forward.

## Process

Follow these steps in order. Each step produces a markdown document in your goal workspace (`workspace/`). Do not skip steps. Do not move to the next step without the human's approval.

### Step 1: Capture the Goal (`goal.md`)

Ask the human to describe their goal. What do they want to achieve? Why? What does success look like?

Write their answer into the workspace `goal.md`. Keep it in their voice. Don't embellish.

### Step 2: Capture the Description (`description.md`)

Ask the human to describe the project. How does it work? What are the components? What technology choices have they made?

Write their answer into the workspace `description.md`. Structure it clearly but stay faithful to what they said. Ask clarifying questions if something is ambiguous.

### Step 3: Research (`research.md`)

Based on the goal and description, research:
- Existing tools and projects in the same space
- Relevant libraries, frameworks, and patterns
- Technical approaches and trade-offs
- Potential pitfalls

Present findings in the workspace `research.md`. Let the human review before proceeding.

### Step 4: Plan (`plan.md`)

Based on goal, description, and research, propose an implementation plan:
- What to build first
- What components exist
- How they connect
- What the milestones are

Write to the workspace `plan.md`. The human reviews and may revise.

### Step 5: Review (`review.md`)

Critically review the plan:
- What are the risks?
- What's missing?
- What's over-engineered?
- What should change?

Write the review to the workspace `review.md`. Discuss with the human.

### Step 6: Revise the Plan

Update the workspace `plan.md` based on the review. The human approves the final plan.

### Step 7: Task Decomposition

Use the `create_task` tool (available via the `work` MCP server) to write each task directly. Call it once per task based on your full knowledge of the plan — do not re-read workspace files. After creating all tasks, run `work tree` to verify the dependency graph.

```bash
work tree
```

### Step 8: Choose Execution Mode

Ask the human how they want to execute the tasks:

**Option A — Inline (current agent):** You work through each task sequentially in this conversation, in dependency order. Use `work ready` to find the next task, do the work, then mark it complete with `work complete <id>`. Best for small goals (≤5 tasks, no parallelism needed).

**Option B — Worktrees (separate agents):** Each task runs as an isolated Claude Code session in its own git worktree. Best for large goals, parallel work, or tasks that benefit from isolation.

```bash
work ready        # show tasks with no pending dependencies
work run <id>     # launch a separate agent for a task (Option B)
work complete <id> # mark a task done (Option A)
```

If the human chooses **Option A**, invoke `/work-execute` and work through ready tasks one by one, marking each complete before moving to the next.

## Rules

- One step at a time. Don't rush ahead.
- The human steers. You execute.
- Ask questions when unsure. Don't assume.
- Keep documents concise. No filler.
- Each document should stand on its own — readable without the others.
