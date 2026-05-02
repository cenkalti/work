package work

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	"github.com/spf13/cobra"
)

func migrateSpaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate-space",
		Short: "Move existing per-task and root workspaces to ~/.work/space/<project>/",
		Long: `Idempotent migration. Moves:
  - <repo>/.work/space/<task>/  →  ~/.work/space/<project>/<task>/
  - <repo>/workspace/ (real dir) →  ~/.work/space/<project>/_root/   (symlinked back at <repo>/workspace)

Worktree workspace symlinks are repointed to the new locations. Re-running is a no-op.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := detectLocation(cmd)
			if err != nil {
				return err
			}
			if !loc.IsRoot() {
				return fmt.Errorf("must be run from the repo root")
			}
			return runMigrateSpace(loc.RootRepo)
		},
	}
}

func runMigrateSpace(root string) error {
	if _, err := paths.EnsureProject(root); err != nil {
		return err
	}

	moved, repointed, skipped, err := migrateTaskWorkspaces(root)
	if err != nil {
		return err
	}

	rootMigrated, err := migrateRootWorkspace(root)
	if err != nil {
		return err
	}

	rootStr := "no"
	if rootMigrated {
		rootStr = "yes"
	}
	fmt.Printf("moved %d task workspaces, repointed %d symlinks, migrated root workspace? %s, skipped %d\n",
		moved, repointed, rootStr, skipped)
	return nil
}

func migrateTaskWorkspaces(root string) (moved, repointed, skipped int, err error) {
	oldRoot := filepath.Join(root, ".work", "space")
	entries, err := os.ReadDir(oldRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, fmt.Errorf("reading old space dir: %w", err)
	}

	worktreeLinks, err := collectWorktreeWorkspaceLinks(root)
	if err != nil {
		return 0, 0, 0, err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		task := e.Name()
		// Skip "." / ".." defensively, and any dotfile entry (e.g. an
		// orphan .git, .DS_Store) — those aren't task workspaces.
		if task == "." || task == ".." || strings.HasPrefix(task, ".") {
			continue
		}
		oldPath := filepath.Join(oldRoot, task)
		newPath := paths.Workspace(root, task)

		if _, err := os.Stat(newPath); err == nil {
			fmt.Fprintf(os.Stderr, "skip %s: destination already exists at %s\n", task, newPath)
			skipped++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
			return moved, repointed, skipped, fmt.Errorf("creating parent for %s: %w", task, err)
		}
		if err := moveDir(oldPath, newPath); err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", task, err)
			skipped++
			continue
		}
		moved++

		// Repoint any worktree workspace symlink that pointed at oldPath.
		for wtPath, target := range worktreeLinks {
			if target != oldPath {
				continue
			}
			link := paths.WorkspaceLink(wtPath)
			if err := os.Remove(link); err != nil {
				fmt.Fprintf(os.Stderr, "warn: removing stale symlink at %s: %v\n", link, err)
				continue
			}
			if err := os.Symlink(newPath, link); err != nil {
				fmt.Fprintf(os.Stderr, "warn: recreating symlink at %s: %v\n", link, err)
				continue
			}
			repointed++
		}
	}
	return moved, repointed, skipped, nil
}

func migrateRootWorkspace(root string) (bool, error) {
	link := paths.WorkspaceLink(root)
	info, err := os.Lstat(link)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", link, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, nil // already a symlink
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s exists and is neither a symlink nor a directory", link)
	}

	dst, err := paths.RootWorkspace(root)
	if err != nil {
		return false, err
	}
	if entries, err := os.ReadDir(dst); err == nil && len(entries) > 0 {
		fmt.Fprintf(os.Stderr, "skip root workspace: destination %s is non-empty\n", dst)
		return false, nil
	} else if err == nil {
		// dst exists but empty — remove so rename can succeed
		if err := os.Remove(dst); err != nil {
			return false, fmt.Errorf("clearing empty destination %s: %w", dst, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("checking destination %s: %w", dst, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return false, fmt.Errorf("creating parent for root workspace: %w", err)
	}
	if err := moveDir(link, dst); err != nil {
		return false, fmt.Errorf("moving root workspace: %w", err)
	}
	if err := os.Symlink(dst, link); err != nil {
		return false, fmt.Errorf("creating root workspace symlink: %w", err)
	}
	return true, nil
}

// collectWorktreeWorkspaceLinks returns a map of worktree path → resolved
// `workspace` symlink target for every worktree under <root>/.work/tree.
// Entries whose `workspace` is missing or not a symlink are omitted.
func collectWorktreeWorkspaceLinks(root string) (map[string]string, error) {
	wtRoot := paths.WorktreeRoot(root)
	entries, err := os.ReadDir(wtRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading worktree root: %w", err)
	}
	links := make(map[string]string, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wtPath := filepath.Join(wtRoot, e.Name())
		link := paths.WorkspaceLink(wtPath)
		target, err := os.Readlink(link)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(wtPath, target)
		}
		target = filepath.Clean(target)
		links[wtPath] = target
	}
	return links, nil
}

// moveDir performs os.Rename. Both paths in this codebase live under $HOME,
// so cross-device moves shouldn't occur in practice; if they ever do, the
// error surfaces and the user fixes it.
func moveDir(src, dst string) error {
	return os.Rename(src, dst)
}
