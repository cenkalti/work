package todo

import (
	"os"
	"strings"
	"testing"
)

func TestParseRoundTripBasic(t *testing.T) {
	buf, err := os.ReadFile("testdata/render_basic.md")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 3 {
		t.Fatalf("expected 3 top-level items, got %d", len(parsed))
	}
	root := parsed[0]
	if root.ID != "a3f2b9" || root.Title != "Fix kube reconnect bug" || root.Status != StatusActive {
		t.Fatalf("root mismatch: %+v", root)
	}
	if len(root.Links) != 1 || root.Links[0].Label != "issue" {
		t.Fatalf("links mismatch: %+v", root.Links)
	}
	if len(root.Projects) != 1 || root.Projects[0] != "~/projects/teleport" {
		t.Fatalf("projects mismatch: %+v", root.Projects)
	}
	if root.Notes != "Tried bumping keepalive, no effect.\nMore notes here." {
		t.Fatalf("notes mismatch: %q", root.Notes)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}
	if root.Children[0].ID != "x7k4qm" || root.Children[0].Status != StatusActive {
		t.Fatalf("child 0 mismatch: %+v", root.Children[0])
	}
	if root.Children[1].ID != "q4w7zb" || root.Children[1].Status != StatusCancelled {
		t.Fatalf("child 1 mismatch: %+v", root.Children[1])
	}
	if parsed[1].ID != "h5r1tj" || parsed[1].Status != StatusCompleted {
		t.Fatalf("h5r1tj mismatch: %+v", parsed[1])
	}
	if parsed[2].ID != "m9b3vc" || parsed[2].Status != StatusOpen {
		t.Fatalf("m9b3vc mismatch: %+v", parsed[2])
	}
	if len(parsed[2].Links) != 1 || parsed[2].Links[0].Label != "" || parsed[2].Links[0].URL != "https://dentist.example.com" {
		t.Fatalf("dentist link mismatch: %+v", parsed[2].Links)
	}
}

func TestParseRoundTripEmpty(t *testing.T) {
	buf, err := os.ReadFile("testdata/render_empty.md")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 0 {
		t.Fatalf("expected empty, got %d", len(parsed))
	}
}

func TestParseRenderRoundTripFixedPoint(t *testing.T) {
	todos, order := basicFixture()
	first := Render(todos, order)
	parsed, err := Parse(first)
	if err != nil {
		t.Fatal(err)
	}
	// Reconstruct todos+order from parsed and re-render.
	rebuilt := make(map[string]*Todo)
	rebuiltOrder := make([]string, 0, len(parsed))
	var walk func(p *ParsedTodo)
	walk = func(p *ParsedTodo) {
		td := &Todo{
			ID:       p.ID,
			Title:    p.Title,
			Status:   p.Status,
			Links:    p.Links,
			Projects: p.Projects,
			Notes:    p.Notes,
		}
		for _, c := range p.Children {
			td.Children = append(td.Children, c.ID)
			walk(c)
		}
		rebuilt[p.ID] = td
	}
	for _, p := range parsed {
		rebuiltOrder = append(rebuiltOrder, p.ID)
		walk(p)
	}
	second := Render(rebuilt, rebuiltOrder)
	if string(first) != string(second) {
		t.Fatalf("round-trip not a fixed point:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestParseEmptyTitleRejected(t *testing.T) {
	_, err := Parse([]byte("- [ ]  <!--abc123-->\n"))
	if err == nil || !strings.Contains(err.Error(), "empty title") {
		t.Fatalf("expected empty title error, got %v", err)
	}
}

func TestParseDuplicateIDRejected(t *testing.T) {
	buf := []byte("- [ ] one <!--aaaaaa-->\n- [ ] two <!--aaaaaa-->\n")
	_, err := Parse(buf)
	if err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestParseIndentJumpRejected(t *testing.T) {
	buf := []byte("- [ ] root <!--aaaaaa-->\n    - [ ] grandchild <!--bbbbbb-->\n")
	_, err := Parse(buf)
	if err == nil || !strings.Contains(err.Error(), "skips a level") {
		t.Fatalf("expected skip-level error, got %v", err)
	}
}

func TestParseBareURLLink(t *testing.T) {
	buf := []byte("- [ ] item <!--aaaaaa-->\n  @ https://example.com\n")
	parsed, err := Parse(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed[0].Links) != 1 || parsed[0].Links[0].Label != "" || parsed[0].Links[0].URL != "https://example.com" {
		t.Fatalf("bare URL link mismatch: %+v", parsed[0].Links)
	}
}

func TestParseHandTypedIDAccepted(t *testing.T) {
	buf := []byte("- [ ] manually pinned <!--zzzzzz-->\n")
	parsed, err := Parse(buf)
	if err != nil {
		t.Fatalf("expected no error for hand-typed id, got %v", err)
	}
	if parsed[0].ID != "zzzzzz" {
		t.Fatalf("id mismatch: %q", parsed[0].ID)
	}
}

func TestParseOrphanMetadataRejected(t *testing.T) {
	buf := []byte("  @ https://orphan.example\n")
	_, err := Parse(buf)
	if err == nil || !strings.Contains(err.Error(), "no parent item") {
		t.Fatalf("expected orphan error, got %v", err)
	}
}
