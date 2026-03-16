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

// Location holds detected information about the current working location.
type Location struct {
	RootRepo string
	Branch   string // empty when at the root repo (not inside any task worktree)
}

// IsRoot reports whether the current location is the root repo.
func (l *Location) IsRoot() bool {
	return l.Branch == ""
}

// TasksDir returns the subtasks directory for the current branch.
func (l *Location) TasksDir() string {
	return paths.TasksDir(l.RootRepo, l.Branch)
}

// WorktreePath returns the path to a named worktree.
func (l *Location) WorktreePath(branch string) string {
	return paths.Worktree(l.RootRepo, branch)
}

// WorktreeRoot returns the root directory containing all worktrees.
func (l *Location) WorktreeRoot() string {
	return paths.WorktreeRoot(l.RootRepo)
}

// ResolveName converts a name to a full branch name.
// If name contains a dot, it's used as-is (absolute).
// If inside a task worktree, the name is treated as a child task ID appended to the current branch.
// Otherwise it's a root task name.
func (l *Location) ResolveName(name string) string {
	if strings.Contains(name, ".") {
		return name
	}
	if !l.IsRoot() {
		return l.Branch + "." + name
	}
	return name
}

// ResolveBranch returns explicit if non-empty, otherwise the current branch.
// Returns an error if at the root repo with no explicit branch.
func (l *Location) ResolveBranch(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if !l.IsRoot() {
		return l.Branch, nil
	}
	return "", fmt.Errorf("not in a task worktree; specify a task explicitly")
}

// Detect determines the current working context by examining
// the working directory and current branch.
func Detect() (*Location, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootRepo := resolveRootRepo(cwd)
	loc := &Location{RootRepo: rootRepo}
	if cwd != rootRepo {
		branch, err := git.CurrentBranch(cwd)
		if err != nil {
			return nil, fmt.Errorf("detect current branch: %w", err)
		}
		loc.Branch = branch
	}
	return loc, nil
}

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
