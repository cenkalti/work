package location

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/paths"
)

// Type represents where the user is running the CLI from.
type Type int

const (
	Root Type = iota // root repo, main/master, or unknown branch
	Goal             // goal worktree
	Task             // task worktree
)

// Location holds detected information about the current working location.
type Location struct {
	Type     Type
	RootRepo string // always set
	Goal     string // set for LocationGoal and LocationTask
	Task     string // set for LocationTask only
}

// Detect determines the current working context by examining
// the working directory, root repo, and current branch.
func Detect() (*Location, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootRepo := resolveRootRepo(cwd)
	branch, err := git.CurrentBranch(cwd)
	if err != nil {
		return nil, fmt.Errorf("detect current branch: %w", err)
	}

	ctx := &Location{RootRepo: rootRepo}
	ctx.Type, ctx.Goal, ctx.Task = classifyBranch(branch, rootRepo == cwd)
	return ctx, nil
}

// classifyBranch determines location, goal branch, and task ID from a git branch name.
// atRoot should be true when the cwd equals the root repo (i.e. not inside a worktree subdir).
func classifyBranch(branch string, atRoot bool) (Type, string, string) {
	if goalBranch, taskID, ok := strings.Cut(branch, "."); ok {
		return Task, goalBranch, taskID
	}
	if !atRoot {
		return Goal, branch, ""
	}
	return Root, "", ""
}

// GoalSpacePath returns the path to the goal's workspace directory.
func (c *Location) GoalSpacePath() string {
	if c.Goal == "" {
		return ""
	}
	return paths.GoalWorkspace(c.RootRepo, c.Goal)
}

// TasksDir returns the path to the goal's tasks directory.
func (c *Location) TasksDir() string {
	return filepath.Join(c.GoalSpacePath(), "tasks")
}

// WorktreePath returns the path to a named worktree.
func (c *Location) WorktreePath(branch string) string {
	return filepath.Join(paths.WorktreeRoot(c.RootRepo), branch)
}

// WorktreeRoot returns the root directory containing all worktrees.
func (c *Location) WorktreeRoot() string {
	return paths.WorktreeRoot(c.RootRepo)
}

// ResolveName splits a dot-notation name into goal and taskID.
// If name contains a dot, it's goal.task. If no dot and we're in a goal/task
// worktree, it's treated as a task ID under the current goal. Otherwise it's a goal.
// taskID is empty when name resolves to a goal.
func (c *Location) ResolveName(name string) (goal, task string) {
	if goal, task, ok := strings.Cut(name, "."); ok {
		return goal, task
	}
	if c.Type == Goal || c.Type == Task {
		return c.Goal, name
	}
	return name, ""
}

// ResolveGoal returns the goal branch, using explicit if provided, else inferring
// from the current worktree context.
func (c *Location) ResolveGoal(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if c.Goal != "" {
		return c.Goal, nil
	}
	return "", fmt.Errorf("not in a goal worktree; specify a goal explicitly")
}

// resolveRootRepo returns the main worktree path (the root repo).
func resolveRootRepo(repo string) string {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return repo
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repo, gitDir)
	}
	return filepath.Dir(gitDir)
}
