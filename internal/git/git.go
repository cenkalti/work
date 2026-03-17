package git

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// CreateWorktree creates a git worktree at the given path on the given branch.
// If the worktree already exists, it is reused. If the branch already exists,
// the worktree is created from it. Returns true if a new worktree was created.
func CreateWorktree(repo, wtPath, branch, startPoint string) (created bool, err error) {
	// If the worktree directory already exists, reuse it.
	if _, err := os.Stat(wtPath); err == nil {
		return false, nil
	}

	// Try creating a new branch; if it already exists, check out the existing one.
	args := []string{"worktree", "add", "-b", branch, wtPath}
	if startPoint != "" {
		args = append(args, startPoint)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		cmd2 := exec.Command("git", "worktree", "add", "-f", wtPath, branch)
		cmd2.Dir = repo
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return false, fmt.Errorf("git worktree add: %s: %w (also tried existing branch: %s: %v)", string(out), err, string(out2), err2)
		}
	}

	return true, nil
}

// RemoveWorktree removes a git worktree by path.
func RemoveWorktree(repo, worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", string(out), err)
	}
	return nil
}

// DefaultBranch returns "main" if it exists, otherwise "master".
func DefaultBranch(repo string) string {
	cmd := exec.Command("git", "rev-parse", "--verify", "main")
	cmd.Dir = repo
	if err := cmd.Run(); err == nil {
		return "main"
	}
	return "master"
}

// CurrentBranch returns the current branch name of the repo.
func CurrentBranch(repo string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("not on a named branch")
	}
	return branch, nil
}

// ListWorktrees returns the paths of all git worktrees.
func ListWorktrees(repo string) ([]string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %s: %w", string(out), err)
	}
	var paths []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// RenameBranch renames a local git branch.
func RenameBranch(repo, oldName, newName string) error {
	cmd := exec.Command("git", "branch", "-m", oldName, newName)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch rename: %s: %w", string(out), err)
	}
	return nil
}

// MoveWorktree moves a git worktree to a new path.
func MoveWorktree(repo, oldPath, newPath string) error {
	cmd := exec.Command("git", "worktree", "move", oldPath, newPath)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree move: %s: %w", string(out), err)
	}
	return nil
}

// DeleteBranch deletes a local git branch.
func DeleteBranch(repo, branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch delete: %s: %w", string(out), err)
	}
	return nil
}

// RemoveWorktreeIfExists removes a git worktree by path, succeeding silently if it does not exist.
func RemoveWorktreeIfExists(repo, worktreePath string) error {
	paths, err := ListWorktrees(repo)
	if err != nil {
		return err
	}
	if slices.Contains(paths, worktreePath) {
		return RemoveWorktree(repo, worktreePath)
	}
	return nil
}

// DeleteBranchIfExists deletes a local git branch, succeeding silently if it does not exist.
func DeleteBranchIfExists(repo, branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repo
	if err := cmd.Run(); err != nil {
		return nil
	}
	return DeleteBranch(repo, branch)
}
