package domain

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree is a directory under <repo>/.work/tree/<name>/.
//
// Worktree.Name is the basename of the worktree directory (filesystem
// identity). It may diverge from the checked-out git branch name; use
// Worktree.Branch to query the current branch from git.
type Worktree struct {
	RepoPath string
	Name     string
}

// Repo returns the parent Repo.
func (w Worktree) Repo() Repo {
	return Repo{Path: w.RepoPath}
}

// Path is the worktree directory: <repo>/.work/tree/<name>.
func (w Worktree) Path() string {
	return filepath.Join(w.RepoPath, ".work", "tree", w.Name)
}

// WorkspacePath is the per-task workspace directory:
// $HOME/.work/space/<project>/<name>.
func (w Worktree) WorkspacePath() string {
	wr, err := WorkspaceRoot()
	if err != nil {
		// Fall back to a sentinel that fails loudly when used.
		return filepath.Join("/invalid-workspace-root", w.Repo().ProjectName(), w.Name)
	}
	return filepath.Join(wr, w.Repo().ProjectName(), w.Name)
}

// TasksDir is the subtasks directory inside the workspace.
func (w Worktree) TasksDir() string {
	return filepath.Join(w.WorkspacePath(), "tasks")
}

// WorkspaceLink is the path to the workspace symlink inside the worktree.
func (w Worktree) WorkspaceLink() string {
	return filepath.Join(w.Path(), "workspace")
}

// Branch returns the currently checked-out branch by querying git. The
// returned Branch.Name may diverge from Worktree.Name.
func (w Worktree) Branch() (Branch, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = w.Path()
	out, err := cmd.Output()
	if err != nil {
		return Branch{}, fmt.Errorf("git rev-parse --abbrev-ref HEAD: %w", err)
	}
	return Branch{
		RepoPath: w.RepoPath,
		Name:     strings.TrimSpace(string(out)),
	}, nil
}
