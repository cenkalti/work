package todo

import (
	"os"
	"strings"
	"testing"
)

func basicFixture() (map[string]*Todo, []string) {
	todos := map[string]*Todo{
		"a3f2b9": {
			ID:     "a3f2b9",
			Title:  "Fix kube reconnect bug",
			Status: StatusActive,
			Links:  []Link{{Label: "issue", URL: "https://github.com/example/issue/1"}},
			Projects: []string{
				"~/projects/teleport",
			},
			Notes:    "Tried bumping keepalive, no effect.\nMore notes here.",
			Children: []string{"x7k4qm", "q4w7zb"},
		},
		"x7k4qm": {ID: "x7k4qm", Title: "reproduce locally", Status: StatusActive},
		"q4w7zb": {ID: "q4w7zb", Title: "poke at flaky test", Status: StatusCancelled},
		"h5r1tj": {ID: "h5r1tj", Title: "Ship marketplace owner", Status: StatusCompleted},
		"m9b3vc": {
			ID:     "m9b3vc",
			Title:  "Call dentist",
			Status: StatusOpen,
			Links:  []Link{{URL: "https://dentist.example.com"}},
		},
	}
	order := []string{"a3f2b9", "h5r1tj", "m9b3vc"}
	return todos, order
}

func TestRenderBasicGolden(t *testing.T) {
	todos, order := basicFixture()
	got := Render(todos, order)
	want, err := os.ReadFile("testdata/render_basic.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("render mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderEmptyGolden(t *testing.T) {
	got := Render(map[string]*Todo{}, []string{})
	want, err := os.ReadFile("testdata/render_empty.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("render mismatch:\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestRenderLinkLabelOptional(t *testing.T) {
	todos := map[string]*Todo{
		"aaaaaa": {ID: "aaaaaa", Title: "labeled", Status: StatusOpen, Links: []Link{{Label: "x", URL: "https://x"}}},
		"bbbbbb": {ID: "bbbbbb", Title: "bare", Status: StatusOpen, Links: []Link{{URL: "https://y"}}},
	}
	got := string(Render(todos, []string{"aaaaaa", "bbbbbb"}))
	if !strings.Contains(got, "@ [x](https://x)") {
		t.Fatalf("expected labeled link, got:\n%s", got)
	}
	if !strings.Contains(got, "@ https://y") {
		t.Fatalf("expected bare link, got:\n%s", got)
	}
	if strings.Contains(got, "@ [](https://y)") {
		t.Fatalf("bare link should not have empty brackets, got:\n%s", got)
	}
}

func TestRenderIndentationTwoSpacesPerLevel(t *testing.T) {
	todos := map[string]*Todo{
		"aaaaaa": {ID: "aaaaaa", Title: "root", Status: StatusOpen, Children: []string{"bbbbbb"}},
		"bbbbbb": {ID: "bbbbbb", Title: "child", Status: StatusOpen, Children: []string{"cccccc"}},
		"cccccc": {ID: "cccccc", Title: "grandchild", Status: StatusOpen},
	}
	got := string(Render(todos, []string{"aaaaaa"}))
	if !strings.Contains(got, "\n- [ ] root <!--aaaaaa-->\n") {
		t.Fatalf("root indentation wrong:\n%s", got)
	}
	if !strings.Contains(got, "\n  - [ ] child <!--bbbbbb-->\n") {
		t.Fatalf("child indentation wrong:\n%s", got)
	}
	if !strings.Contains(got, "\n    - [ ] grandchild <!--cccccc-->\n") {
		t.Fatalf("grandchild indentation wrong:\n%s", got)
	}
}
