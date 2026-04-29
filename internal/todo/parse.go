package todo

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// ParsedTodo is the in-memory result of parsing the editor buffer. It
// carries the same fields as Todo plus a tree of children and the source
// line for error messages.
type ParsedTodo struct {
	ID       string
	Title    string
	Status   string
	Links    []Link
	Projects []string
	Notes    string
	Children []*ParsedTodo
	Line     int
}

type lineKind int

const (
	kindBlank lineKind = iota
	kindItem
	kindLink
	kindProject
	kindNote
)

// Parse converts the editor buffer back into a tree of ParsedTodos in user
// order. Returns an error with line numbers if the buffer is malformed.
func Parse(buf []byte) ([]*ParsedTodo, error) {
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	type frame struct {
		depth int
		item  *ParsedTodo
	}

	var top []*ParsedTodo
	var stack []frame
	seenIDs := make(map[string]int)
	current := (*ParsedTodo)(nil)
	currentDepth := -1
	notesByItem := make(map[*ParsedTodo]*strings.Builder)
	headerDone := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		line := strings.TrimRight(raw, " \t")

		indent, rest := splitIndent(line)

		if !headerDone {
			if rest == "" || strings.HasPrefix(rest, "<!--") {
				continue
			}
			headerDone = true
		}

		if rest == "" {
			continue
		}

		kind, marker, body := classify(rest)

		if kind == kindItem {
			if indent%2 != 0 {
				return nil, fmt.Errorf("line %d: indentation must be a multiple of 2 spaces", lineNum)
			}
			depth := indent / 2

			if !validMarker(marker) {
				return nil, fmt.Errorf("line %d: unknown status marker %q (expected one of ' ', '/', 'x', '-')", lineNum, string(marker))
			}

			title, id, err := splitTitleAndID(body)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			if title == "" {
				return nil, fmt.Errorf("line %d: empty title", lineNum)
			}

			for len(stack) > 0 && stack[len(stack)-1].depth >= depth {
				stack = stack[:len(stack)-1]
			}
			parentDepth := -1
			if len(stack) > 0 {
				parentDepth = stack[len(stack)-1].depth
			}
			if depth > parentDepth+1 {
				return nil, fmt.Errorf("line %d: indent skips a level", lineNum)
			}

			pt := &ParsedTodo{
				ID:     id,
				Title:  title,
				Status: statusFromMarker(marker),
				Line:   lineNum,
			}
			if id != "" {
				if prev, ok := seenIDs[id]; ok {
					return nil, fmt.Errorf("line %d: duplicate id %q (first seen on line %d)", lineNum, id, prev)
				}
				seenIDs[id] = lineNum
			}

			if len(stack) == 0 {
				top = append(top, pt)
			} else {
				parent := stack[len(stack)-1].item
				parent.Children = append(parent.Children, pt)
			}
			stack = append(stack, frame{depth: depth, item: pt})
			current = pt
			currentDepth = depth
			continue
		}

		// Metadata line. Must follow an item, at item.depth+1 worth of indent.
		if current == nil {
			return nil, fmt.Errorf("line %d: %q has no parent item", lineNum, raw)
		}
		expectedIndent := (currentDepth + 1) * 2
		if indent != expectedIndent {
			return nil, fmt.Errorf("line %d: metadata indent mismatch (got %d, want %d)", lineNum, indent, expectedIndent)
		}

		switch kind {
		case kindLink:
			current.Links = append(current.Links, parseLink(body))
		case kindProject:
			current.Projects = append(current.Projects, body)
		case kindNote:
			b, ok := notesByItem[current]
			if !ok {
				b = &strings.Builder{}
				notesByItem[current] = b
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(body)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for item, b := range notesByItem {
		item.Notes = b.String()
	}

	return top, nil
}

func splitIndent(line string) (int, string) {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	return i, line[i:]
}

// classify inspects the non-indented portion of a line and returns its kind.
// For kindItem, marker holds the status character and body holds the rest of
// the line (title + optional id comment). For other kinds, body is the
// payload after the sigil.
func classify(rest string) (kind lineKind, marker byte, body string) {
	// Item: "- [X]" with optional " body" tail.
	if len(rest) >= 5 && rest[0] == '-' && rest[1] == ' ' && rest[2] == '[' && rest[4] == ']' {
		switch {
		case len(rest) == 5:
			return kindItem, rest[3], ""
		case rest[5] == ' ':
			return kindItem, rest[3], rest[6:]
		}
	}
	if strings.HasPrefix(rest, "@ ") {
		return kindLink, 0, rest[2:]
	}
	if strings.HasPrefix(rest, "& ") {
		return kindProject, 0, rest[2:]
	}
	return kindNote, 0, rest
}

// splitTitleAndID extracts the optional <!--id--> suffix from an item body
// and returns the trimmed title plus the id (or "" if absent).
func splitTitleAndID(body string) (string, string, error) {
	const open = " <!--"
	const closeSuffix = "-->"
	idx := strings.LastIndex(body, open)
	if idx < 0 || !strings.HasSuffix(body, closeSuffix) {
		return strings.TrimSpace(body), "", nil
	}
	idStart := idx + len(open)
	idEnd := len(body) - len(closeSuffix)
	if idEnd-idStart != idLen {
		return strings.TrimSpace(body), "", nil
	}
	id := body[idStart:idEnd]
	if !isValidID(id) {
		return strings.TrimSpace(body), "", nil
	}
	return strings.TrimSpace(body[:idx]), id, nil
}

func isValidID(s string) bool {
	if len(s) != idLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func validMarker(m byte) bool {
	return m == ' ' || m == '/' || m == 'x' || m == '-'
}

func statusFromMarker(m byte) string {
	switch m {
	case '/':
		return StatusActive
	case 'x':
		return StatusCompleted
	case '-':
		return StatusCancelled
	default:
		return StatusOpen
	}
}

func parseLink(s string) Link {
	if len(s) > 0 && s[0] == '[' {
		closeBracket := strings.Index(s, "](")
		if closeBracket > 0 && strings.HasSuffix(s, ")") {
			return Link{Label: s[1:closeBracket], URL: s[closeBracket+2 : len(s)-1]}
		}
	}
	return Link{URL: s}
}
