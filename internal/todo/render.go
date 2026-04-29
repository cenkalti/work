package todo

import (
	"bytes"
	"fmt"
	"strings"
)

const headerComment = "<!-- todo: [ ] open  [/] active  [x] done  [-] cancelled  | @ link  & path  | edit and save -->\n"

// Render produces the editor buffer for the given store snapshot.
func Render(todos map[string]*Todo, topOrder []string) []byte {
	var buf bytes.Buffer
	buf.WriteString(headerComment)
	buf.WriteString("\n")
	for _, id := range topOrder {
		t, ok := todos[id]
		if !ok {
			continue
		}
		renderItem(&buf, todos, t, 0)
	}
	return buf.Bytes()
}

func renderItem(buf *bytes.Buffer, todos map[string]*Todo, t *Todo, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(buf, "%s- [%s] %s <!--%s-->\n", indent, marker(t.Status), t.Title, t.ID)
	childIndent := strings.Repeat("  ", depth+1)
	for _, l := range t.Links {
		if l.Label != "" {
			fmt.Fprintf(buf, "%s@ [%s](%s)\n", childIndent, l.Label, l.URL)
		} else {
			fmt.Fprintf(buf, "%s@ %s\n", childIndent, l.URL)
		}
	}
	for _, p := range t.Projects {
		fmt.Fprintf(buf, "%s& %s\n", childIndent, p)
	}
	if t.Notes != "" {
		for line := range strings.SplitSeq(t.Notes, "\n") {
			fmt.Fprintf(buf, "%s%s\n", childIndent, line)
		}
	}
	for _, cid := range t.Children {
		c, ok := todos[cid]
		if !ok {
			continue
		}
		renderItem(buf, todos, c, depth+1)
	}
}

func marker(status string) string {
	switch status {
	case StatusActive:
		return "/"
	case StatusCompleted:
		return "x"
	case StatusCancelled:
		return "-"
	default:
		return " "
	}
}
