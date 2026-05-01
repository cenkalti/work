package agent

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTempHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
}

func newTestRecord(t *testing.T) *Record {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid.NewV7: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return &Record{
		ID:           id.String(),
		Name:         "test-agent",
		Project:      "work",
		ProjectRoot:  "/tmp/work",
		TaskID:       "test-agent",
		Branch:       "test-agent",
		WorktreePath: "/tmp/work/.work/tree/test-agent",
		Status:       StatusIdle,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestReadWriteRoundTrip(t *testing.T) {
	setupTempHome(t)

	want := newTestRecord(t)
	want.PaneID = "42"
	want.CurrentSessionID = "abc-123"
	want.MessageCount = 5
	want.LastPromptPreview = "hello world"

	if err := Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(want.ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name || got.PaneID != want.PaneID ||
		got.CurrentSessionID != want.CurrentSessionID || got.Status != want.Status ||
		got.MessageCount != want.MessageCount || got.LastPromptPreview != want.LastPromptPreview ||
		!got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("round-trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestList(t *testing.T) {
	setupTempHome(t)

	a := newTestRecord(t)
	a.WorktreePath = "/tmp/a"
	b := newTestRecord(t)
	b.WorktreePath = "/tmp/b"

	if err := Write(a); err != nil {
		t.Fatal(err)
	}
	if err := Write(b); err != nil {
		t.Fatal(err)
	}

	all, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 records, got %d", len(all))
	}
}

func TestListMissingDirReturnsEmpty(t *testing.T) {
	setupTempHome(t)
	all, err := List()
	if err != nil {
		t.Fatalf("List on missing dir: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("want 0 records, got %d", len(all))
	}
}

func TestFindByWorktree(t *testing.T) {
	setupTempHome(t)

	a := newTestRecord(t)
	a.WorktreePath = "/tmp/a"
	b := newTestRecord(t)
	b.WorktreePath = "/tmp/b"
	if err := Write(a); err != nil {
		t.Fatal(err)
	}
	if err := Write(b); err != nil {
		t.Fatal(err)
	}

	got, err := FindByWorktree("/tmp/b")
	if err != nil {
		t.Fatalf("FindByWorktree: %v", err)
	}
	if got.ID != b.ID {
		t.Errorf("want %s, got %s", b.ID, got.ID)
	}

	_, err = FindByWorktree("/tmp/nope")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want fs.ErrNotExist, got %v", err)
	}
}

func TestFindBySession(t *testing.T) {
	setupTempHome(t)

	a := newTestRecord(t)
	a.CurrentSessionID = "sess-a"
	b := newTestRecord(t)
	b.CurrentSessionID = "sess-b"
	if err := Write(a); err != nil {
		t.Fatal(err)
	}
	if err := Write(b); err != nil {
		t.Fatal(err)
	}

	got, err := FindBySession("sess-b")
	if err != nil {
		t.Fatalf("FindBySession: %v", err)
	}
	if got.ID != b.ID {
		t.Errorf("want %s, got %s", b.ID, got.ID)
	}

	_, err = FindBySession("nope")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want fs.ErrNotExist, got %v", err)
	}

	_, err = FindBySession("")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want fs.ErrNotExist for empty session, got %v", err)
	}
}

func TestAtomicWriteNoTornFile(t *testing.T) {
	setupTempHome(t)

	r := newTestRecord(t)
	if err := Write(r); err != nil {
		t.Fatal(err)
	}

	// Verify there are no leftover temp files in the agents dir.
	dir, _ := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".tmp" {
			t.Errorf("leftover tempfile: %s", name)
		}
	}
}

func TestDelete(t *testing.T) {
	setupTempHome(t)

	r := newTestRecord(t)
	if err := Write(r); err != nil {
		t.Fatal(err)
	}
	if err := Delete(r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(r.ID); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want fs.ErrNotExist, got %v", err)
	}
	// Deleting again is a no-op.
	if err := Delete(r.ID); err != nil {
		t.Errorf("Delete on missing: %v", err)
	}
}
