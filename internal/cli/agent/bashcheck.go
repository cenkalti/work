package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
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

Respond with a JSON object matching the provided schema.`

type hookOutput struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

func runBashCheck(command string, stdout io.Writer) error {
	if command == "" {
		return writeDecision(stdout, "ask", "no command found in hook input")
	}

	decision, reason, err := classifyCommand(command)
	if err != nil {
		return writeDecision(stdout, "ask", "classification error: "+err.Error())
	}

	return writeDecision(stdout, decision, reason)
}

func classifyCommand(command string) (string, string, error) {
	schemaJSON, err := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"decision": map[string]any{
				"type": "string",
				"enum": []string{"allow", "ask"},
			},
			"reason": map[string]any{"type": "string"},
		},
		"required":             []string{"decision", "reason"},
		"additionalProperties": false,
	})
	if err != nil {
		return "", "", fmt.Errorf("marshal schema: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"-p",
		"--model", "claude-haiku-4-5",
		"--output-format", "json",
		"--json-schema", string(schemaJSON),
		"--append-system-prompt", bashClassifyPrompt,
		"--setting-sources", "",
		"--tools", "",
		"--no-session-persistence",
		"Command: "+command,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("claude: %w: %s", err, stderr.String())
	}

	var env struct {
		StructuredOutput struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		} `json:"structured_output"`
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return "", "", fmt.Errorf("decode claude output: %w", err)
	}
	if env.IsError {
		return "", "", fmt.Errorf("claude error: %s", env.Result)
	}

	decision := env.StructuredOutput.Decision
	if decision != "allow" && decision != "ask" {
		return "ask", "unexpected decision: " + decision, nil
	}
	return decision, env.StructuredOutput.Reason, nil
}

func writeDecision(w io.Writer, decision, reason string) error {
	out := hookOutput{}
	out.HookSpecificOutput.HookEventName = "PreToolUse"
	out.HookSpecificOutput.PermissionDecision = decision
	out.HookSpecificOutput.PermissionDecisionReason = reason
	return json.NewEncoder(w).Encode(out)
}
