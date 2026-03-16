package location

import "testing"

func TestClassifyBranch(t *testing.T) {
	tests := []struct {
		branch   string
		atRoot   bool
		wantLoc  Type
		wantGoal string
		wantTask string
	}{
		// root location
		{"main", true, Root, "", ""},
		{"master", true, Root, "", ""},
		{"", true, Root, "", ""},

		// goal worktree (not at root, no dot)
		{"my-goal", false, Goal, "my-goal", ""},
		{"fix-auth", false, Goal, "fix-auth", ""},

		// task worktree (branch contains a dot)
		{"my-goal.my-task", false, Task, "my-goal", "my-task"},
		{"my-goal.my-task", true, Task, "my-goal", "my-task"},
		{"goal.task-with-hyphens", false, Task, "goal", "task-with-hyphens"},

		// edge: multiple dots — only first dot splits
		{"goal.task.extra", false, Task, "goal", "task.extra"},
	}

	for _, tc := range tests {
		loc, goal, taskID := classifyBranch(tc.branch, tc.atRoot)
		if loc != tc.wantLoc {
			t.Errorf("classifyBranch(%q, %v): location = %v, want %v", tc.branch, tc.atRoot, loc, tc.wantLoc)
		}
		if goal != tc.wantGoal {
			t.Errorf("classifyBranch(%q, %v): goal = %q, want %q", tc.branch, tc.atRoot, goal, tc.wantGoal)
		}
		if taskID != tc.wantTask {
			t.Errorf("classifyBranch(%q, %v): taskID = %q, want %q", tc.branch, tc.atRoot, taskID, tc.wantTask)
		}
	}
}

func TestResolveName(t *testing.T) {
	tests := []struct {
		name       string
		location   Type
		goalBranch string
		wantGoal   string
		wantTask   string
	}{
		// explicit goal.task notation
		{"my-goal.my-task", Root, "", "my-goal", "my-task"},
		// shorthand task ID from within a goal worktree
		{"my-task", Goal, "my-goal", "my-goal", "my-task"},
		// shorthand task ID from within a task worktree
		{"other-task", Task, "my-goal", "my-goal", "other-task"},
		// goal name from root — taskID is empty
		{"my-goal", Root, "", "my-goal", ""},
	}

	for _, tc := range tests {
		ctx := &Location{Type: tc.location, Goal: tc.goalBranch}
		goal, taskID := ctx.ResolveName(tc.name)
		if goal != tc.wantGoal || taskID != tc.wantTask {
			t.Errorf("ResolveName(%q) with loc=%v goal=%q: got (%q, %q), want (%q, %q)",
				tc.name, tc.location, tc.goalBranch,
				goal, taskID,
				tc.wantGoal, tc.wantTask)
		}
	}
}
