package location

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// Detect determines the current working context from the worktree directory name.
// The task identity is derived from the worktree's path under <root>/.work/tree/,
// not from the git branch name (which may diverge from the worktree name).
func Detect() (*Location, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootRepo := resolveRootRepo(cwd)
	loc := &Location{RootRepo: rootRepo}

	top, err := worktreeTopLevel(cwd)
	if err != nil {
		return nil, err
	}
	if top == rootRepo {
		return loc, nil
	}

	wtRoot := paths.WorktreeRoot(rootRepo)
	if resolved, err := filepath.EvalSymlinks(wtRoot); err == nil {
		wtRoot = resolved
	}
	if resolved, err := filepath.EvalSymlinks(top); err == nil {
		top = resolved
	}
	name, ok := strings.CutPrefix(top, wtRoot+string(filepath.Separator))
	if !ok {
		return nil, fmt.Errorf("worktree %s is not under %s", top, wtRoot)
	}
	loc.Branch = name
	return loc, nil
}

func worktreeTopLevel(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
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
