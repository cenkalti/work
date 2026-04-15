package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const bashClassifyPrompt = `Classify this bash command to decide whether it needs user approval. Return permissionDecision 'allow' or 'ask' based on these rules, evaluated in order:

1. ALWAYS ALLOW (read-only): Commands that only read state — ls, cat, git status, git log, git diff, grep, find, echo, pwd, which, env, ps, top, df, du, head, tail, wc, file, stat, uname, whoami, id, date, rg, fd, jq, yq, less, more, tree, go doc, go vet, go list, curl/wget with no method flag or explicit GET.
2. ALWAYS ALLOW (restarts): Commands that restart a service or process — systemctl restart, docker restart, kubectl rollout restart, process signals like HUP for reload.
3. ALWAYS ALLOW (new resources): Commands that create new resources without overwriting existing ones — touch, mkdir, git branch, git checkout -b, docker create, kubectl create, helm install (new release), tee to a new file.
4. ALWAYS ALLOW (git staging): git add, git stage — these only modify the index and are easily reversible.
5. ALWAYS ALLOW (build/run): Commands that build or run code — go build, go run, go test, go mod tidy, npm test, npm run, make (without 'clean' or 'install' targets), cargo build, cargo run, cargo test, python script.py, docker build.
6. ALWAYS ALLOW (kubernetes apply): kubectl apply — standard K8s workflow command.
7. ASK (git commit): git commit must ALWAYS be 'ask' — the user wants control over when to commit and what the commit message says.
8. ASK (git checkout without -b): git checkout or git switch to an existing branch — can discard uncommitted changes. Only allow git checkout -b / git switch -c (new branch creation).
9. ASK (git stash): git stash and git stash pop/apply/drop — ask for all stash operations.
10. ASK (git pull): git pull modifies the local branch. git fetch is read-only and allowed under rule 1.
11. ASK (overwrite/upgrade): Commands that overwrite existing resources or upgrade packages — cp over existing files, helm upgrade (including --install), apt upgrade, npm update, go get -u, pip install --upgrade, git push, git rebase, '>' redirect to files (overwrite). '>>' append to files is allowed.
12. ASK (make clean/install): make clean, make install — these are destructive or system-modifying.
13. ASK (non-GET HTTP requests): curl/wget with -X POST, -X PUT, -X DELETE, -d, --data, or any flag implying a non-GET request.
14. ALLOW (test resource deletion): If a delete/destroy/remove command targets something that appears to be a test resource (the name or path contains 'test', 'tmp', 'temp', 'mock', 'fixture', 'fake', 'dummy', 'scratch', 'experimental'), allow it. EXCEPTION: kubectl delete namespace — always ask, even for test namespaces.
15. ASK (GitHub write operations): Any gh command that creates, modifies, or comments on GitHub resources — gh pr create, gh pr edit, gh pr comment, gh pr merge, gh pr close, gh pr review, gh issue create, gh issue edit, gh issue comment, gh issue close, gh release create. Read-only gh commands (gh pr list, gh pr view, gh issue list, gh issue view, gh pr checks, gh pr diff) are allowed under rule 1.
16. ASK (all other destructive): Any command that modifies state not covered above — rm, mv, chmod, chown, git reset, git push --force, docker rm, kubectl delete (non-test), package removal.

Compound commands: For pipes (|), chains (&&, ;, ||), and subshells, evaluate ALL components. If any component would be 'ask', the whole command is 'ask'.
When in doubt, return 'ask'.

Use the classify_command tool to return your decision.`

func bashCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "bash-check",
		Short:  "Classify a bash command for the PreToolUse hook",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBashCheck(os.Stdin, os.Stdout)
		},
	}
}

type hookInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

type hookOutput struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

func runBashCheck(stdin io.Reader, stdout io.Writer) error {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return writeDecision(stdout, "ask", "failed to read stdin: "+err.Error())
	}

	var hook hookInput
	if err := json.Unmarshal(input, &hook); err != nil {
		return writeDecision(stdout, "ask", "failed to parse hook input: "+err.Error())
	}

	command := hook.ToolInput.Command
	if command == "" {
		return writeDecision(stdout, "ask", "no command found in hook input")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = loadEnvValue("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return writeDecision(stdout, "ask", "ANTHROPIC_API_KEY not set")
	}

	decision, reason, err := classifyCommand(apiKey, command)
	if err != nil {
		return writeDecision(stdout, "ask", "classification error: "+err.Error())
	}

	return writeDecision(stdout, decision, reason)
}

func classifyCommand(apiKey, command string) (string, string, error) {
	reqBody := map[string]any{
		"model":      "claude-haiku-4-5",
		"max_tokens": 256,
		"messages": []map[string]string{
			{"role": "user", "content": "Command: " + command},
		},
		"system": bashClassifyPrompt,
		"tool_choice": map[string]string{
			"type": "tool",
			"name": "classify_command",
		},
		"tools": []map[string]any{
			{
				"name":        "classify_command",
				"description": "Return the classification decision for a bash command.",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"decision": map[string]any{
							"type":        "string",
							"enum":        []string{"allow", "ask"},
							"description": "Whether to allow the command or ask the user.",
						},
						"reason": map[string]any{
							"type":        "string",
							"description": "Brief explanation of why.",
						},
					},
					"required": []string{"decision", "reason"},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Type  string          `json:"type"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	for _, block := range apiResp.Content {
		if block.Type != "tool_use" {
			continue
		}
		var result struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if err := json.Unmarshal(block.Input, &result); err != nil {
			return "", "", fmt.Errorf("parse tool input: %w", err)
		}
		if result.Decision != "allow" && result.Decision != "ask" {
			return "ask", "unexpected decision: " + result.Decision, nil
		}
		return result.Decision, result.Reason, nil
	}

	return "", "", fmt.Errorf("no tool_use block in response")
}

func loadEnvValue(key string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(home, "projects", "work", ".env"))
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

func writeDecision(w io.Writer, decision, reason string) error {
	out := hookOutput{}
	out.HookSpecificOutput.HookEventName = "PreToolUse"
	out.HookSpecificOutput.PermissionDecision = decision
	out.HookSpecificOutput.PermissionDecisionReason = reason
	return json.NewEncoder(w).Encode(out)
}
