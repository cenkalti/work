package location

import "testing"

func TestResolveName(t *testing.T) {
	tests := []struct {
		name       string
		branch     string // current branch (empty = root repo)
		wantBranch string
	}{
		// absolute reference (contains dot) → used as-is
		{"my-task.sub-task", "", "my-task.sub-task"},
		{"my-task.sub-task", "my-task", "my-task.sub-task"},
		// root task from root repo
		{"my-task", "", "my-task"},
		// child task from within a task worktree
		{"sub-task", "my-task", "my-task.sub-task"},
		// deeper nesting
		{"leaf", "my-task.sub-task", "my-task.sub-task.leaf"},
	}

	for _, tc := range tests {
		loc := &Location{Branch: tc.branch}
		got := loc.ResolveName(tc.name)
		if got != tc.wantBranch {
			t.Errorf("ResolveName(%q) with branch=%q: got %q, want %q",
				tc.name, tc.branch, got, tc.wantBranch)
		}
	}
}

func TestResolveBranch(t *testing.T) {
	tests := []struct {
		explicit string
		branch   string
		want     string
		wantErr  bool
	}{
		{"explicit", "", "explicit", false},
		{"explicit", "current", "explicit", false},
		{"", "current", "current", false},
		{"", "", "", true},
	}

	for _, tc := range tests {
		loc := &Location{Branch: tc.branch}
		got, err := loc.ResolveBranch(tc.explicit)
		if (err != nil) != tc.wantErr {
			t.Errorf("ResolveBranch(%q) with branch=%q: err=%v, wantErr=%v",
				tc.explicit, tc.branch, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("ResolveBranch(%q) with branch=%q: got %q, want %q",
				tc.explicit, tc.branch, got, tc.want)
		}
	}
}
