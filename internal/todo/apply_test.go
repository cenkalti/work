package todo

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"time"
)

func loadSnapshot(t *testing.T, dir string) (map[string]*Todo, []string) {
	t.Helper()
	todos, order, err := LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	return todos, order
}

func TestApplyNewItemGetsGeneratedID(t *testing.T) {
	dir := t.TempDir()
	parsed := []*ParsedTodo{{Title: "new item", Status: StatusOpen}}
	if err := Apply(dir, parsed, map[string]*Todo{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if parsed[0].ID == "" {
		t.Fatalf("id not assigned")
	}
	re := regexp.MustCompile(`^[a-z0-9]{6}$`)
	if !re.MatchString(parsed[0].ID) {
		t.Fatalf("id %q not 6-char alnum", parsed[0].ID)
	}
	if _, err := os.Stat(filepath.Join(dir, parsed[0].ID+".json")); err != nil {
		t.Fatalf("expected JSON file written: %v", err)
	}
}

func TestApplyRemovedItemTrashed(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "killme", Title: "x", Status: StatusOpen})
	if err := WriteOrder(dir, []string{"killme"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)
	if err := Apply(dir, nil, snap, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "killme.json")); !os.IsNotExist(err) {
		t.Fatalf("expected open file gone")
	}
	if _, err := os.Stat(filepath.Join(dir, trashDir, "killme.json")); err != nil {
		t.Fatalf("expected file in trash: %v", err)
	}
}

func TestApplyTitleChangeRewrites(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "rename", Title: "old", Status: StatusOpen})
	if err := WriteOrder(dir, []string{"rename"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)
	parsed := []*ParsedTodo{{ID: "rename", Title: "new", Status: StatusOpen}}
	if err := Apply(dir, parsed, snap, time.Now()); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir, "rename")
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "new" {
		t.Fatalf("title not rewritten: %q", out.Title)
	}
}

func TestApplyStatusTransitionOpenToCompletedSetsClosedAt(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "tswtch", Title: "x", Status: StatusOpen})
	if err := WriteOrder(dir, []string{"tswtch"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	parsed := []*ParsedTodo{{ID: "tswtch", Title: "x", Status: StatusCompleted}}
	if err := Apply(dir, parsed, snap, now); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir, "tswtch")
	if err != nil {
		t.Fatal(err)
	}
	if out.ClosedAt == nil || !out.ClosedAt.Equal(now) {
		t.Fatalf("ClosedAt not set: %v", out.ClosedAt)
	}
}

func TestApplyStatusTransitionCompletedToOpenClearsClosedAt(t *testing.T) {
	dir := t.TempDir()
	old := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	writeTodo(t, dir, &Todo{ID: "reopen", Title: "x", Status: StatusCompleted, ClosedAt: ptrTime(old)})
	if err := WriteOrder(dir, []string{"reopen"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)
	parsed := []*ParsedTodo{{ID: "reopen", Title: "x", Status: StatusOpen}}
	if err := Apply(dir, parsed, snap, time.Now()); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir, "reopen")
	if err != nil {
		t.Fatal(err)
	}
	if out.ClosedAt != nil {
		t.Fatalf("expected nil ClosedAt, got %v", out.ClosedAt)
	}
}

func TestApplyStatusTransitionCompletedToCancelledKeepsClosedAt(t *testing.T) {
	dir := t.TempDir()
	old := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	writeTodo(t, dir, &Todo{ID: "swtch2", Title: "x", Status: StatusCompleted, ClosedAt: ptrTime(old)})
	if err := WriteOrder(dir, []string{"swtch2"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)
	parsed := []*ParsedTodo{{ID: "swtch2", Title: "x", Status: StatusCancelled}}
	if err := Apply(dir, parsed, snap, time.Now()); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir, "swtch2")
	if err != nil {
		t.Fatal(err)
	}
	if out.ClosedAt == nil || !out.ClosedAt.Equal(old) {
		t.Fatalf("expected ClosedAt unchanged at %v, got %v", old, out.ClosedAt)
	}
}

func TestApplyTopLevelReorderRewritesOrder(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "first1", Title: "first", Status: StatusOpen})
	writeTodo(t, dir, &Todo{ID: "secnd2", Title: "second", Status: StatusOpen})
	if err := WriteOrder(dir, []string{"first1", "secnd2"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)
	parsed := []*ParsedTodo{
		{ID: "secnd2", Title: "second", Status: StatusOpen},
		{ID: "first1", Title: "first", Status: StatusOpen},
	}
	if err := Apply(dir, parsed, snap, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, order, err := LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(order, []string{"secnd2", "first1"}) {
		t.Fatalf("expected reorder, got %v", order)
	}
}

func TestApplyReparentingUpdatesBothParents(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "alpha1", Title: "alpha", Status: StatusOpen, Children: []string{"chldab"}})
	writeTodo(t, dir, &Todo{ID: "betaaa", Title: "beta", Status: StatusOpen})
	writeTodo(t, dir, &Todo{ID: "chldab", Title: "child", Status: StatusOpen})
	if err := WriteOrder(dir, []string{"alpha1", "betaaa"}); err != nil {
		t.Fatal(err)
	}
	snap, _ := loadSnapshot(t, dir)

	parsed := []*ParsedTodo{
		{ID: "alpha1", Title: "alpha", Status: StatusOpen},
		{ID: "betaaa", Title: "beta", Status: StatusOpen, Children: []*ParsedTodo{
			{ID: "chldab", Title: "child", Status: StatusOpen},
		}},
	}
	if err := Apply(dir, parsed, snap, time.Now()); err != nil {
		t.Fatal(err)
	}
	alpha, _ := Load(dir, "alpha1")
	beta, _ := Load(dir, "betaaa")
	if len(alpha.Children) != 0 {
		t.Fatalf("expected alpha to lose child, got %v", alpha.Children)
	}
	if !slices.Equal(beta.Children, []string{"chldab"}) {
		t.Fatalf("expected beta to gain child, got %v", beta.Children)
	}
}

func TestApplyHandTypedIDForUnknownCreatesFile(t *testing.T) {
	dir := t.TempDir()
	parsed := []*ParsedTodo{{ID: "pinned", Title: "pinned item", Status: StatusOpen}}
	if err := Apply(dir, parsed, map[string]*Todo{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pinned.json")); err != nil {
		t.Fatalf("expected pinned file: %v", err)
	}
}
