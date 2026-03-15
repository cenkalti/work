package cli

import "testing"

func TestClassifyBranch(t *testing.T) {
	tests := []struct {
		branch   string
		atRoot   bool
		wantLoc  LocationType
		wantGoal string
		wantTask string
	}{
		// root location
		{"main", true, LocationRoot, "", ""},
		{"master", true, LocationRoot, "", ""},
		{"", true, LocationRoot, "", ""},

		// goal worktree (not at root, no dot)
		{"my-goal", false, LocationGoal, "my-goal", ""},
		{"fix-auth", false, LocationGoal, "fix-auth", ""},

		// task worktree (branch contains a dot)
		{"my-goal.my-task", false, LocationTask, "my-goal", "my-task"},
		{"my-goal.my-task", true, LocationTask, "my-goal", "my-task"},
		{"goal.task-with-hyphens", false, LocationTask, "goal", "task-with-hyphens"},

		// edge: multiple dots — only first dot splits
		{"goal.task.extra", false, LocationTask, "goal", "task.extra"},
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
		name      string
		location  LocationType
		goalBranch string
		wantGoal  string
		wantTask  string
		wantIsTask bool
	}{
		// explicit goal.task notation
		{"my-goal.my-task", LocationRoot, "", "my-goal", "my-task", true},
		// shorthand task ID from within a goal worktree
		{"my-task", LocationGoal, "my-goal", "my-goal", "my-task", true},
		// shorthand task ID from within a task worktree
		{"other-task", LocationTask, "my-goal", "my-goal", "other-task", true},
		// goal name from root
		{"my-goal", LocationRoot, "", "my-goal", "", false},
	}

	for _, tc := range tests {
		ctx := &WorkContext{Location: tc.location, GoalBranch: tc.goalBranch}
		goal, taskID, isTask := ctx.ResolveName(tc.name)
		if goal != tc.wantGoal || taskID != tc.wantTask || isTask != tc.wantIsTask {
			t.Errorf("ResolveName(%q) with loc=%v goal=%q: got (%q, %q, %v), want (%q, %q, %v)",
				tc.name, tc.location, tc.goalBranch,
				goal, taskID, isTask,
				tc.wantGoal, tc.wantTask, tc.wantIsTask)
		}
	}
}
