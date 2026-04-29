package todo

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	todopkg "github.com/cenkalti/work/internal/todo"
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo",
		Short: "Edit a global todo list in $EDITOR",
		RunE:  run,
	}
	cmd.CompletionOptions.HiddenDefaultCmd = true
	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	dir, err := todopkg.Dir()
	if err != nil {
		return err
	}
	if err := todopkg.EnsureDir(dir); err != nil {
		return err
	}

	release, err := acquireLock(dir)
	if err != nil {
		return err
	}
	defer release()

	if err := todopkg.ArchiveSweep(dir, time.Now()); err != nil {
		return err
	}

	snapshot, order, err := todopkg.LoadAll(dir)
	if err != nil {
		return err
	}

	original := todopkg.Render(snapshot, order)
	originalHash := sha256.Sum256(original)

	tmp, err := os.CreateTemp(dir, "edit-*.md")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	leftover, err := absorbLeftover(dir, tmpPath)
	if err != nil {
		return err
	}
	initial := original
	if leftover != nil {
		initial = leftover
	}
	if err := os.WriteFile(tmpPath, initial, 0o644); err != nil {
		return err
	}

	if err := launchEditor(tmpPath); err != nil {
		return err
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}
	if sha256.Sum256(edited) == originalHash {
		return nil
	}

	if isEffectivelyEmpty(edited) {
		fmt.Fprintln(os.Stderr, "todo: empty buffer, aborting")
		return nil
	}

	parsed, err := todopkg.Parse(edited)
	if err != nil {
		recoverPath := filepath.Join(dir, fmt.Sprintf(".recover-%d.md", time.Now().Unix()))
		if werr := os.WriteFile(recoverPath, edited, 0o644); werr != nil {
			return fmt.Errorf("%w (also failed to write recovery file: %v)", err, werr)
		}
		fmt.Fprintf(os.Stderr, "todo: %v\n", err)
		fmt.Fprintf(os.Stderr, "todo: buffer saved to %s\n", recoverPath)
		os.Exit(1)
	}

	if err := todopkg.Apply(dir, parsed, snapshot, time.Now()); err != nil {
		return err
	}

	return todopkg.ArchiveSweep(dir, time.Now())
}

func acquireLock(dir string) (func(), error) {
	lockPath := filepath.Join(dir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			fmt.Fprintln(os.Stderr, "todo: another todo session is running")
			os.Exit(1)
		}
		return nil, err
	}
	return func() { f.Close() }, nil
}

func launchEditor(path string) error {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return fmt.Errorf("$EDITOR is not set; set it to your preferred editor (e.g. EDITOR=nvim)")
	}
	c := exec.Command("sh", "-c", fmt.Sprintf(`%s "$1"`, editor), "sh", path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}
	return nil
}

// absorbLeftover looks for previously-orphaned edit-*.md tempfiles in the
// store dir (skipping ourTmp). If found, returns the contents of the most
// recent one and deletes all leftovers. The lock guarantees no other todo
// process is touching these files, so any leftover is from a prior crash.
func absorbLeftover(dir, ourTmp string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var leftovers []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "edit-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		if filepath.Join(dir, name) == ourTmp {
			continue
		}
		leftovers = append(leftovers, e)
	}
	if len(leftovers) == 0 {
		return nil, nil
	}

	type stamped struct {
		path string
		mod  time.Time
	}
	stamps := make([]stamped, 0, len(leftovers))
	for _, e := range leftovers {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		stamps = append(stamps, stamped{path: filepath.Join(dir, e.Name()), mod: info.ModTime()})
	}
	mostRecent := stamps[0]
	for _, s := range stamps[1:] {
		if s.mod.After(mostRecent.mod) {
			mostRecent = s
		}
	}

	data, err := os.ReadFile(mostRecent.path)
	if err != nil {
		return nil, err
	}
	for _, s := range stamps {
		os.Remove(s.path)
	}
	fmt.Fprintf(os.Stderr, "todo: recovered leftover buffer from %s (modified %s)\n",
		filepath.Base(mostRecent.path), mostRecent.mod.Format(time.RFC3339))
	return data, nil
}

// isEffectivelyEmpty returns true if the buffer contains nothing but
// whitespace and HTML comment lines (the header). Mirrors `git commit`
// aborting when the message body is empty.
func isEffectivelyEmpty(buf []byte) bool {
	for line := range bytes.SplitSeq(buf, []byte("\n")) {
		t := bytes.TrimSpace(line)
		if len(t) == 0 {
			continue
		}
		if bytes.HasPrefix(t, []byte("<!--")) {
			continue
		}
		return false
	}
	return true
}
