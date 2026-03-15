package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCreateAndRemove(t *testing.T) {
	repo := t.TempDir()
	run(t, repo, "git", "init")
	run(t, repo, "git", "commit", "--allow-empty", "-m", "init")

	branch := "cenk.alti.test1"
	wtPath := filepath.Join(repo, ".work", "tree", branch)
	created, err := CreateWorktree(repo, wtPath, branch, "")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatal("worktree directory not created")
	}

	if !created {
		t.Error("expected created = true for new worktree")
	}

	if err := RemoveWorktree(repo, wtPath); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Fatal("worktree directory still exists after remove")
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
}
