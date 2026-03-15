package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cenkalti/work/internal/git"
	"github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

type workContextKey struct{}

// LocationType represents where the user is running the CLI from.
type LocationType int

const (
	LocationRoot LocationType = iota // root repo, main/master, or unknown branch
	LocationGoal                     // goal worktree
	LocationTask                     // task worktree
)

// WorkContext holds detected information about the current working location.
type WorkContext struct {
	Location   LocationType
	RootRepo   string // always set
	GoalBranch string // set for LocationGoal and LocationTask
	TaskID     string // set for LocationTask only
}

// detectContext determines the current working context by examining
// the working directory, root repo, and current branch.
func detectContext() (*WorkContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootRepo := resolveRootRepo(cwd)
	branch, err := git.CurrentBranch(cwd)
	if err != nil {
		return nil, fmt.Errorf("detect current branch: %w", err)
	}

	ctx := &WorkContext{RootRepo: rootRepo}
	ctx.Location, ctx.GoalBranch, ctx.TaskID = classifyBranch(branch, rootRepo == cwd)
	return ctx, nil
}

// classifyBranch determines location, goal branch, and task ID from a git branch name.
// atRoot should be true when the cwd equals the root repo (i.e. not inside a worktree subdir).
func classifyBranch(branch string, atRoot bool) (LocationType, string, string) {
	if goalBranch, taskID, ok := strings.Cut(branch, "."); ok {
		return LocationTask, goalBranch, taskID
	}
	if !atRoot {
		return LocationGoal, branch, ""
	}
	return LocationRoot, "", ""
}

// persistWorkContext calls detectContext and stores the result in the command's context.
func persistWorkContext(cmd *cobra.Command, args []string) error {
	wc, err := detectContext()
	if err != nil {
		return err
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return nil
}

// workContext retrieves the *WorkContext stored by PersistentPreRunE.
// If not yet stored (e.g. during shell completion), it detects and caches it.
// Exits the program if context detection fails.
func workContext(cmd *cobra.Command) *WorkContext {
	if wc, ok := cmd.Context().Value(workContextKey{}).(*WorkContext); ok {
		return wc
	}
	wc, err := detectContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return wc
}

// GoalSpacePath returns the path to the goal's workspace directory.
// Returns empty string if not in a goal or task context.
func (c *WorkContext) GoalSpacePath() string {
	if c.GoalBranch == "" {
		return ""
	}
	return GoalSpacePath(c.RootRepo, c.GoalBranch)
}

// TasksDir returns the path to the goal's tasks directory.
// Returns empty string if not in a goal or task context.
func (c *WorkContext) TasksDir() string {
	return filepath.Join(c.GoalSpacePath(), "tasks")
}

// WorktreePath returns the path to a named worktree.
func (c *WorkContext) WorktreePath(branch string) string {
	return filepath.Join(c.WorktreeRoot(), branch)
}

func (c *WorkContext) WorktreeRoot() string {
	return WorktreeRootPath(c.RootRepo)
}

// ResolveName splits a dot-notation name into goal and taskID.
// If name contains a dot, it's goal.task. If no dot and we're in a goal/task
// worktree, it's treated as a task ID under the current goal. Otherwise it's a goal.
func (c *WorkContext) ResolveName(name string) (goal, taskID string, isTask bool) {
	if goalBranch, tid, ok := strings.Cut(name, "."); ok {
		return goalBranch, tid, true
	}
	if c.Location == LocationGoal || c.Location == LocationTask {
		return c.GoalBranch, name, true
	}
	return name, "", false
}

// ResolveGoal returns the goal branch, using explicit if provided, else inferring
// from the current worktree context.
func (c *WorkContext) ResolveGoal(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if c.GoalBranch != "" {
		return c.GoalBranch, nil
	}
	return "", fmt.Errorf("not in a goal worktree; specify a goal explicitly")
}

// goalSpacePath returns the path to a goal's workspace.
func goalSpacePath(rootRepo, goalBranch string) string {
	return GoalSpacePath(rootRepo, goalBranch)
}

// taskSpacePath returns the path to a task agent's workspace.
func taskSpacePath(rootRepo, goalBranch, taskID string) string {
	return TaskSpacePath(rootRepo, goalBranch, taskID)
}

// tasksDirFor returns the tasks directory for a given goal.
func tasksDirFor(rootRepo, goalBranch string) string {
	return filepath.Join(GoalSpacePath(rootRepo, goalBranch), "tasks")
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

// listGoalWorktreeNames returns worktree names that are goals (no dots).
func listGoalWorktreeNames(rootRepo string) []string {
	var goals []string
	wtRoot := WorktreeRootPath(rootRepo)
	for _, name := range listWorktreeNames(wtRoot) {
		if !strings.Contains(name, ".") {
			goals = append(goals, name)
		}
	}
	return goals
}

// listTaskIDsFiltered returns task IDs matching an optional filter.
func listTaskIDsFiltered(tasksDir string, filter func(*task.Task) bool) []string {
	tasks, err := task.LoadAll(tasksDir)
	if err != nil {
		return nil
	}
	var ids []string
	for id, t := range tasks {
		if filter == nil || filter(t) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}
