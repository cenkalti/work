package domain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Detect resolves the current working location from CWD.
//
// Repo is always returned; Worktree is nil when CWD is the root repo (not
// inside any task worktree). Branch is not resolved here — callers that
// need the checked-out branch should call Worktree.Branch on demand.
func Detect() (Repo, *Worktree, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Repo{}, nil, err
	}
	rootPath := resolveRootRepo(cwd)
	repo := Repo{Path: rootPath}

	top, err := worktreeTopLevel(cwd)
	if err != nil {
		return repo, nil, nil
	}
	if top == rootPath {
		return repo, nil, nil
	}

	wtRoot := repo.WorktreeRoot()
	if resolved, err := filepath.EvalSymlinks(wtRoot); err == nil {
		wtRoot = resolved
	}
	if resolved, err := filepath.EvalSymlinks(top); err == nil {
		top = resolved
	}
	name, ok := strings.CutPrefix(top, wtRoot+string(filepath.Separator))
	if !ok {
		return Repo{}, nil, fmt.Errorf("worktree %s is not under %s", top, wtRoot)
	}
	return repo, &Worktree{RepoPath: rootPath, Name: name}, nil
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
