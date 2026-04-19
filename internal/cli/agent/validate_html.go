package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type validationIssue struct {
	Line     int
	Col      int
	Severity string
	Message  string
}

type validationReport struct {
	Valid bool
	HTML  []validationIssue
	JS    []validationIssue
	CSS   []validationIssue
}

var (
	scriptRe = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
	styleRe  = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)
)

func validateHTMLFile(ctx context.Context, path string) validationReport {
	content, err := os.ReadFile(path)
	if err != nil {
		return validationReport{HTML: []validationIssue{{Severity: "error", Message: fmt.Sprintf("cannot read %s: %v", path, err)}}}
	}

	html := runHTMLValidate(ctx, path)
	js := validateInlineJS(ctx, content)
	css := validateInlineCSS(content)

	valid := errorCount(html) == 0 && errorCount(js) == 0 && errorCount(css) == 0
	return validationReport{Valid: valid, HTML: html, JS: js, CSS: css}
}

func errorCount(issues []validationIssue) int {
	n := 0
	for _, i := range issues {
		if i.Severity == "error" {
			n++
		}
	}
	return n
}

func runHTMLValidate(ctx context.Context, path string) []validationIssue {
	cmd := exec.CommandContext(ctx, "npx", "--yes", "html-validate", "--preset=standard", "--formatter=json", path)
	out, err := cmd.Output()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return []validationIssue{{Severity: "warning", Message: fmt.Sprintf("html-validate unavailable: %v", err)}}
	}
	return parseHTMLValidateJSON(out)
}

func parseHTMLValidateJSON(out []byte) []validationIssue {
	var reports []struct {
		Messages []struct {
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity int    `json:"severity"`
			Message  string `json:"message"`
			RuleID   string `json:"ruleId"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &reports); err != nil {
		return []validationIssue{{Severity: "warning", Message: fmt.Sprintf("could not parse html-validate output: %v", err)}}
	}
	var issues []validationIssue
	for _, r := range reports {
		for _, m := range r.Messages {
			sev := "warning"
			if m.Severity >= 2 {
				sev = "error"
			}
			msg := m.Message
			if m.RuleID != "" {
				msg = fmt.Sprintf("%s (%s)", msg, m.RuleID)
			}
			issues = append(issues, validationIssue{Line: m.Line, Col: m.Column, Severity: sev, Message: msg})
		}
	}
	return issues
}

func validateInlineJS(ctx context.Context, content []byte) []validationIssue {
	m := scriptRe.FindSubmatch(content)
	if m == nil {
		return nil
	}
	body := strings.TrimSpace(string(m[1]))
	if body == "" {
		return nil
	}
	tmp, err := os.CreateTemp("", "presentation-*.mjs")
	if err != nil {
		return []validationIssue{{Severity: "warning", Message: fmt.Sprintf("js validator temp file: %v", err)}}
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(body); err != nil {
		tmp.Close()
		return []validationIssue{{Severity: "warning", Message: fmt.Sprintf("js validator write: %v", err)}}
	}
	tmp.Close()

	cmd := exec.CommandContext(ctx, "node", "--check", tmp.Name())
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return []validationIssue{{Severity: "warning", Message: fmt.Sprintf("node unavailable for JS validation: %v", err)}}
	}
	line, col, msg := parseNodeCheckError(string(out))
	return []validationIssue{{Line: line, Col: col, Severity: "error", Message: msg}}
}

var (
	nodeLocRe = regexp.MustCompile(`:(\d+)(?::(\d+))?`)
	nodeErrRe = regexp.MustCompile(`^\w*Error: `)
)

func parseNodeCheckError(out string) (int, int, string) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var line, col int
	for _, l := range lines {
		if m := nodeLocRe.FindStringSubmatch(l); m != nil {
			fmt.Sscanf(m[1], "%d", &line)
			if m[2] != "" {
				fmt.Sscanf(m[2], "%d", &col)
			}
			break
		}
	}
	var msg string
	for _, l := range lines {
		if nodeErrRe.MatchString(l) {
			msg = strings.TrimSpace(l)
			break
		}
	}
	if msg == "" {
		msg = "JS syntax error"
	}
	return line, col, msg
}

func validateInlineCSS(content []byte) []validationIssue {
	m := styleRe.FindSubmatch(content)
	if m == nil {
		return nil
	}
	css := string(m[1])
	opens, closes := countBraces(css)
	if opens == closes {
		return nil
	}
	return []validationIssue{{Severity: "error", Message: fmt.Sprintf("CSS brace mismatch: %d '{' vs %d '}'", opens, closes)}}
}

func countBraces(css string) (int, int) {
	var opens, closes int
	i := 0
	for i < len(css) {
		c := css[i]
		switch {
		case c == '/' && i+1 < len(css) && css[i+1] == '*':
			end := strings.Index(css[i+2:], "*/")
			if end < 0 {
				return opens, closes
			}
			i += end + 4
		case c == '"' || c == '\'':
			quote := c
			i++
			for i < len(css) && css[i] != quote {
				if css[i] == '\\' && i+1 < len(css) {
					i += 2
					continue
				}
				i++
			}
			i++
		case c == '{':
			opens++
			i++
		case c == '}':
			closes++
			i++
		default:
			i++
		}
	}
	return opens, closes
}

func formatReport(r validationReport) string {
	var b strings.Builder
	writeSection := func(name string, issues []validationIssue) {
		errs := errorCount(issues)
		if errs == 0 {
			return
		}
		fmt.Fprintf(&b, "%s: %d error(s)\n", name, errs)
		for _, i := range issues {
			if i.Severity != "error" {
				continue
			}
			loc := ""
			if i.Line > 0 {
				loc = fmt.Sprintf("line %d", i.Line)
				if i.Col > 0 {
					loc += fmt.Sprintf(" col %d", i.Col)
				}
				loc += ": "
			}
			fmt.Fprintf(&b, "  - %s%s\n", loc, i.Message)
		}
	}
	writeSection("HTML", r.HTML)
	writeSection("JS", r.JS)
	writeSection("CSS", r.CSS)
	if b.Len() == 0 {
		return ""
	}
	b.WriteString("\nFix these with Edit and retry. Do not re-run Write.")
	return b.String()
}
