package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestWriteLoadRoundTripNilClosedAt(t *testing.T) {
	dir := t.TempDir()
	in := &Todo{
		ID:     "abc123",
		Title:  "test",
		Status: StatusOpen,
		Links:  []Link{{Label: "x", URL: "https://example.com"}},
	}
	if err := Write(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if out.ClosedAt != nil {
		t.Fatalf("expected nil ClosedAt, got %v", out.ClosedAt)
	}
	if out.Title != "test" || out.Status != StatusOpen {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
	if len(out.Links) != 1 || out.Links[0].URL != "https://example.com" {
		t.Fatalf("links round-trip mismatch: %+v", out.Links)
	}
}

func TestWriteLoadRoundTripNonNilClosedAt(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	in := &Todo{
		ID:       "xyz789",
		Title:    "done",
		Status:   StatusCompleted,
		ClosedAt: &now,
	}
	if err := Write(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir, "xyz789")
	if err != nil {
		t.Fatal(err)
	}
	if out.ClosedAt == nil || !out.ClosedAt.Equal(now) {
		t.Fatalf("ClosedAt round-trip failed: got %v want %v", out.ClosedAt, now)
	}
}

func TestGenerateIDFormat(t *testing.T) {
	dir := t.TempDir()
	re := regexp.MustCompile(`^[a-z0-9]{6}$`)
	for range 100 {
		id, err := GenerateID(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(id) {
			t.Fatalf("id %q does not match pattern", id)
		}
	}
}

func TestGenerateIDAvoidsCollisions(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, archiveDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, trashDir), 0o755); err != nil {
		t.Fatal(err)
	}
	// Plant ids in each location.
	for _, p := range []string{
		filepath.Join(dir, "aaaaaa.json"),
		filepath.Join(dir, archiveDir, "bbbbbb.json"),
		filepath.Join(dir, trashDir, "cccccc.json"),
	} {
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, taken := range []string{"aaaaaa", "bbbbbb", "cccccc"} {
		if !idTaken(dir, taken) {
			t.Fatalf("expected %s to be detected as taken", taken)
		}
	}
	// Generate a few; none should ever match the planted ids.
	for range 50 {
		id, err := GenerateID(dir)
		if err != nil {
			t.Fatal(err)
		}
		if id == "aaaaaa" || id == "bbbbbb" || id == "cccccc" {
			t.Fatalf("collision: generated %s", id)
		}
	}
}

func TestLoadAllNoOrderFileRebuildsByMtimeDesc(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "old111", Status: StatusOpen, Title: "old"})
	writeTodo(t, dir, &Todo{ID: "mid222", Status: StatusOpen, Title: "mid"})
	writeTodo(t, dir, &Todo{ID: "new333", Status: StatusOpen, Title: "new"})

	now := time.Now()
	setMtime(t, filepath.Join(dir, "old111.json"), now.Add(-3*time.Hour))
	setMtime(t, filepath.Join(dir, "mid222.json"), now.Add(-2*time.Hour))
	setMtime(t, filepath.Join(dir, "new333.json"), now.Add(-1*time.Hour))

	stderr := captureStderr(t, func() {
		_, order, err := LoadAll(dir)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"new333", "mid222", "old111"}
		if !slices.Equal(order, want) {
			t.Fatalf("order mismatch: got %v want %v", order, want)
		}
	})
	if !strings.Contains(stderr, "rebuilt") {
		t.Fatalf("expected stderr warning, got %q", stderr)
	}
}

func TestLoadAllOrderDropsMissingIDs(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "real11", Status: StatusOpen, Title: "real"})
	if err := WriteOrder(dir, []string{"ghost1", "real11"}); err != nil {
		t.Fatal(err)
	}
	stderr := captureStderr(t, func() {
		_, order, err := LoadAll(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(order, []string{"real11"}) {
			t.Fatalf("got %v want [real11]", order)
		}
	})
	if !strings.Contains(stderr, "rebuilt") {
		t.Fatalf("expected stderr warning, got %q", stderr)
	}
}

func TestLoadAllOrderAppendsMissingIDsByMtimeDesc(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "first1", Status: StatusOpen, Title: "first"})
	writeTodo(t, dir, &Todo{ID: "extraA", Status: StatusOpen, Title: "extra a"})
	writeTodo(t, dir, &Todo{ID: "extraB", Status: StatusOpen, Title: "extra b"})
	if err := WriteOrder(dir, []string{"first1"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	setMtime(t, filepath.Join(dir, "first1.json"), now.Add(-3*time.Hour))
	setMtime(t, filepath.Join(dir, "extraA.json"), now.Add(-2*time.Hour))
	setMtime(t, filepath.Join(dir, "extraB.json"), now.Add(-1*time.Hour))

	_, order, err := LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"first1", "extraB", "extraA"}
	if !slices.Equal(order, want) {
		t.Fatalf("got %v want %v", order, want)
	}
}

func TestWriteOrderAtomic(t *testing.T) {
	dir := t.TempDir()
	if err := WriteOrder(dir, []string{"abc111", "def222"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, orderFile))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"abc111", "def222"}) {
		t.Fatalf("got %v", got)
	}
	// No leftover .tmp-* files.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Fatalf("leftover tempfile: %s", e.Name())
		}
	}
}

func TestTrashAndArchiveMoveFiles(t *testing.T) {
	dir := t.TempDir()
	writeTodo(t, dir, &Todo{ID: "trashm", Status: StatusOpen, Title: "trash me"})
	writeTodo(t, dir, &Todo{ID: "archvm", Status: StatusCompleted, Title: "archive me"})

	if err := Trash(dir, "trashm"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "trashm.json")); !os.IsNotExist(err) {
		t.Fatalf("expected source removed; err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, trashDir, "trashm.json")); err != nil {
		t.Fatalf("expected file in trash: %v", err)
	}

	if err := Archive(dir, "archvm"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "archvm.json")); !os.IsNotExist(err) {
		t.Fatalf("expected source removed; err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, archiveDir, "archvm.json")); err != nil {
		t.Fatalf("expected file in archive: %v", err)
	}
}

func writeTodo(t *testing.T, dir string, td *Todo) {
	t.Helper()
	if err := Write(dir, td); err != nil {
		t.Fatal(err)
	}
}

func setMtime(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- b.String()
	}()
	fn()
	w.Close()
	os.Stderr = orig
	return <-done
}
