package domain

import (
	"path/filepath"
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
	return filepath.Join(WorkspaceRoot(), w.Repo().ProjectName(), w.Name)
}

// TasksDir is the subtasks directory inside the workspace.
func (w Worktree) TasksDir() string {
	return filepath.Join(w.WorkspacePath(), "tasks")
}

// WorkspaceLink is the path to the workspace symlink inside the worktree.
func (w Worktree) WorkspaceLink() string {
	return filepath.Join(w.Path(), "workspace")
}
