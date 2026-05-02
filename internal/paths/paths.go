package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// workspaces. The whole tree under here is intended to be backed up via git.
func WorkspaceRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving workspace root: %w", err)
	}
	return filepath.Join(home, ".work", "space"), nil
}

// ProjectName returns the project name for a root repo path. Currently just
// the basename. Collisions between two repos sharing a basename are detected
// at workspace-creation time via EnsureProject and a .source marker.
func ProjectName(root string) string {
	return filepath.Base(root)
}

// ProjectDir returns the per-project workspace directory:
// $HOME/.work/space/<project-name>.
func ProjectDir(root string) (string, error) {
	wr, err := WorkspaceRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(wr, ProjectName(root)), nil
}

// EnsureProject creates the project directory if missing and writes a
// .source marker recording the abs path of root. If the marker already
// exists with a different abs path, returns an error to surface the
// collision instead of silently sharing the directory.
func EnsureProject(root string) (string, error) {
	dir, err := ProjectDir(root)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolving root path: %w", err)
	}
	marker := filepath.Join(dir, ".source")
	if existing, err := os.ReadFile(marker); err == nil {
		if strings.TrimSpace(string(existing)) != rootAbs {
			return "", fmt.Errorf(
				"project name %q already taken by %s; rename one of the repos",
				ProjectName(root), strings.TrimSpace(string(existing)),
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

func WorktreeRoot(root string) string {
	return filepath.Join(root, ".work", "tree")
}

func Worktree(root, branch string) string {
	return filepath.Join(root, ".work", "tree", branch)
}

// Workspace returns the workspace directory for a task:
// $HOME/.work/space/<project>/<branch>. Pure path computation; callers
// are responsible for calling EnsureProject before creating files here.
func Workspace(root, branch string) string {
	wr, err := WorkspaceRoot()
	if err != nil {
		// Fall back to a sentinel that will fail loudly when used.
		return filepath.Join("/invalid-workspace-root", ProjectName(root), branch)
	}
	return filepath.Join(wr, ProjectName(root), branch)
}

func TasksDir(root, branch string) string {
	return filepath.Join(Workspace(root, branch), "tasks")
}

// WorkspaceLink returns the workspace symlink path inside a directory.
func WorkspaceLink(dir string) string {
	return filepath.Join(dir, "workspace")
}

// RootWorkspace is the per-project anonymous slot used when planning at the
// repo root: $HOME/.work/space/<project>/_root.
func RootWorkspace(root string) (string, error) {
	dir, err := ProjectDir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "_root"), nil
}

// EnsureRootWorkspace makes the root repo's `workspace/` resolve to
// $HOME/.work/space/<project>/_root so root planning is in the backup tree.
//
// Behavior:
//   - If <root>/workspace already exists (real dir or any symlink), it is
//     left alone — migration of pre-existing real dirs is a separate
//     concern (see `work migrate-space`).
//   - If <root>/workspace does not exist, the project dir + the _root dir
//     are created and a symlink is established.
//
// Returns the resolved root workspace path.
func EnsureRootWorkspace(root string) (string, error) {
	if _, err := EnsureProject(root); err != nil {
		return "", err
	}
	rw, err := RootWorkspace(root)
	if err != nil {
		return "", err
	}
	link := WorkspaceLink(root)
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

// LocalTasksDir returns the tasks directory relative to a working directory (./workspace/tasks).
func LocalTasksDir(cwd string) string {
	return filepath.Join(cwd, "workspace", "tasks")
}

// ParentBranch returns everything before the last dot. Returns "" for root tasks.
func ParentBranch(branch string) string {
	if i := strings.LastIndex(branch, "."); i >= 0 {
		return branch[:i]
	}
	return ""
}

// BranchID returns the last component after the last dot. Returns the full branch for root tasks.
func BranchID(branch string) string {
	if i := strings.LastIndex(branch, "."); i >= 0 {
		return branch[i+1:]
	}
	return branch
}
