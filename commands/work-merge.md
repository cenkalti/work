# Merge

Merge the current task branch into its parent.

## Step 1: Run Tests

Run the test suite and fix any failures before merging.

## Step 2: Merge

```bash
work merge
```

This merges the current branch into its parent branch (or the default branch for root tasks) and streams git output directly.

## Step 3: Verify

Confirm the result looks correct and re-run tests on the parent branch if needed.
