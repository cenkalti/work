package domain

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProjectsDir returns the root directory containing all projects.
// Honors WORK_PROJECTS_DIR, falling back to $HOME/projects.
func ProjectsDir() (string, error) {
	if dir := os.Getenv("WORK_PROJECTS_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving projects dir: %w", err)
	}
	return filepath.Join(home, "projects"), nil
}

// WorkspaceRoot returns $HOME/.work/space, the parent of every project's
// workspaces. The whole tree under here is intended to be backed up.
func WorkspaceRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving workspace root: %w", err)
	}
	return filepath.Join(home, ".work", "space"), nil
}

// LocalTasksDir returns ./workspace/tasks under cwd. Used by `task` CLI
// commands that operate on the workspace symlink without resolving the
// full domain.
func LocalTasksDir(cwd string) string {
	return filepath.Join(cwd, "workspace", "tasks")
}
