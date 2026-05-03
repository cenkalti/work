package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Repo is a git repository on disk. Path is the absolute path to the repo
// root (the parent of the git common dir).
type Repo struct {
	Path string
}

// ProjectName is the basename of the repo path.
func (r Repo) ProjectName() string {
	return filepath.Base(r.Path)
}

// ProjectDir is the per-project workspace directory:
// $HOME/.work/space/<project-name>.
func (r Repo) ProjectDir() (string, error) {
	return filepath.Join(WorkspaceRoot(), r.ProjectName()), nil
}

// WorktreeRoot returns the root directory containing all worktrees:
// <repo>/.work/tree.
func (r Repo) WorktreeRoot() string {
	return filepath.Join(r.Path, ".work", "tree")
}

// WorkspaceLink is the workspace symlink at the repo root.
func (r Repo) WorkspaceLink() string {
	return filepath.Join(r.Path, "workspace")
}

// EnsureProject creates the project directory if missing and writes a
// .source marker recording the abs path of the repo. If the marker exists
// with a different abs path, returns an error so the collision surfaces
// instead of silently sharing the directory.
func (r Repo) EnsureProject() (string, error) {
	dir, err := r.ProjectDir()
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(r.Path)
	if err != nil {
		return "", fmt.Errorf("resolving root path: %w", err)
	}
	marker := filepath.Join(dir, ".source")
	if existing, err := os.ReadFile(marker); err == nil {
		if strings.TrimSpace(string(existing)) != rootAbs {
			return "", fmt.Errorf(
				"project name %q already taken by %s; rename one of the repos",
				r.ProjectName(), strings.TrimSpace(string(existing)),
			)
		}
		return dir, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("reading project source marker: %w", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating project dir: %w", err)
	}
	if err := os.WriteFile(marker, []byte(rootAbs+"\n"), 0644); err != nil {
		return "", fmt.Errorf("writing project source marker: %w", err)
	}
	return dir, nil
}

// EnsureRootWorkspace makes <repo>/workspace resolve to
// $HOME/.work/space/<project>/_root so root planning is in the backup tree.
//
// If <repo>/workspace already exists (real dir or any symlink), it is left
// alone — migration of pre-existing real dirs is a separate concern.
// Otherwise the project dir + the _root dir are created and the symlink
// is established.
func (r Repo) EnsureRootWorkspace() (string, error) {
	dir, err := r.EnsureProject()
	if err != nil {
		return "", err
	}
	rw := filepath.Join(dir, "_root")
	link := r.WorkspaceLink()
	if _, err := os.Lstat(link); err == nil {
		return rw, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("checking root workspace link: %w", err)
	}
	if err := os.MkdirAll(rw, 0755); err != nil {
		return "", fmt.Errorf("creating root workspace dir: %w", err)
	}
	if err := os.Symlink(rw, link); err != nil {
		return "", fmt.Errorf("creating root workspace symlink: %w", err)
	}
	return rw, nil
}
