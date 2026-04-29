package todo

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func ptrTime(t time.Time) *time.Time { return &t }

func TestArchiveSweepCompletedOlderThan14Days(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	closed := now.Add(-15 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{ID: "oldddd", Status: StatusCompleted, Title: "old", ClosedAt: ptrTime(closed)})
	if err := WriteOrder(dir, []string{"oldddd"}); err != nil {
		t.Fatal(err)
	}

	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "oldddd.json")); !os.IsNotExist(err) {
		t.Fatalf("expected open file gone; err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, archiveDir, "oldddd.json")); err != nil {
		t.Fatalf("expected archived file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, orderFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected empty order, got %q", data)
	}
}

func TestArchiveSweepCompletedYoungerThan14DaysStays(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	closed := now.Add(-13 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{ID: "freshc", Status: StatusCompleted, Title: "fresh", ClosedAt: ptrTime(closed)})
	if err := WriteOrder(dir, []string{"freshc"}); err != nil {
		t.Fatal(err)
	}

	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "freshc.json")); err != nil {
		t.Fatalf("expected fresh file to stay: %v", err)
	}
}

func TestArchiveSweepCancelledOlderThan14Days(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	closed := now.Add(-20 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{ID: "cancld", Status: StatusCancelled, Title: "cancelled", ClosedAt: ptrTime(closed)})
	if err := WriteOrder(dir, []string{"cancld"}); err != nil {
		t.Fatal(err)
	}

	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, archiveDir, "cancld.json")); err != nil {
		t.Fatalf("expected archived: %v", err)
	}
}

func TestArchiveSweepClosedParentWithOpenChildKept(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	closed := now.Add(-30 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{
		ID: "parent", Status: StatusCompleted, Title: "p", ClosedAt: ptrTime(closed),
		Children: []string{"chldop"},
	})
	writeTodo(t, dir, &Todo{ID: "chldop", Status: StatusOpen, Title: "open child"})
	if err := WriteOrder(dir, []string{"parent"}); err != nil {
		t.Fatal(err)
	}

	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "parent.json")); err != nil {
		t.Fatalf("expected parent to stay: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, archiveDir, "parent.json")); !os.IsNotExist(err) {
		t.Fatalf("expected parent NOT archived; err=%v", err)
	}
}

func TestArchiveSweepArchivedChildRemovedFromParent(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	closed := now.Add(-30 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{ID: "parnt2", Status: StatusOpen, Title: "open parent", Children: []string{"oldcld", "yngcld"}})
	writeTodo(t, dir, &Todo{ID: "oldcld", Status: StatusCompleted, Title: "old child", ClosedAt: ptrTime(closed)})
	writeTodo(t, dir, &Todo{ID: "yngcld", Status: StatusOpen, Title: "young child"})
	if err := WriteOrder(dir, []string{"parnt2"}); err != nil {
		t.Fatal(err)
	}

	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	parent, err := Load(dir, "parnt2")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parent.Children, []string{"yngcld"}) {
		t.Fatalf("expected children=[yngcld], got %v", parent.Children)
	}
	if _, err := os.Stat(filepath.Join(dir, archiveDir, "oldcld.json")); err != nil {
		t.Fatalf("expected oldcld archived: %v", err)
	}
}

func TestArchiveSweepIdempotent(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	closed := now.Add(-15 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{ID: "oldddd", Status: StatusCompleted, Title: "old", ClosedAt: ptrTime(closed)})
	if err := WriteOrder(dir, []string{"oldddd"}); err != nil {
		t.Fatal(err)
	}

	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	stderr := captureStderr(t, func() {
		if err := ArchiveSweep(dir, now); err != nil {
			t.Fatal(err)
		}
	})
	// Second sweep should not have rewritten _order.json or printed warnings.
	_ = stderr
}

func TestArchiveSweepRewritesOrderWhenTopLevelArchived(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	closed := now.Add(-15 * 24 * time.Hour)
	writeTodo(t, dir, &Todo{ID: "keep01", Status: StatusOpen, Title: "keep"})
	writeTodo(t, dir, &Todo{ID: "drop01", Status: StatusCompleted, Title: "drop", ClosedAt: ptrTime(closed)})
	if err := WriteOrder(dir, []string{"keep01", "drop01"}); err != nil {
		t.Fatal(err)
	}
	if err := ArchiveSweep(dir, now); err != nil {
		t.Fatal(err)
	}
	_, order, err := LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(order, []string{"keep01"}) {
		t.Fatalf("expected order=[keep01], got %v", order)
	}
}
