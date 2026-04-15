package harness

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func surgicalTool(s *server.MCPServer) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("surgical",
		mcp.WithDescription("Check whether a git diff follows the Surgical Changes principle. Every changed line should trace to the user's request. Flags drive-by improvements, style changes, and unrelated edits. Returns a score from 0-10."),
		mcp.WithString("diff",
			mcp.Description("The git diff to analyze. If omitted, uses 'git diff --staged' in the current directory."),
		),
		mcp.WithString("task",
			mcp.Description("The original task or bug description the diff is supposed to implement"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		diff := req.GetString("diff", "")
		task := req.GetString("task", "")

		if diff == "" {
			out, err := exec.CommandContext(ctx, "git", "diff", "--staged").Output()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("git diff --staged: %v", err)), nil
			}
			diff = string(out)
		}
		if strings.TrimSpace(diff) == "" {
			return mcp.NewToolResultText("No diff to analyze (nothing staged)."), nil
		}

		var prompt string
		if task != "" {
			prompt = fmt.Sprintf("Task: %s\n\nDiff:\n%s", task, diff)
		} else {
			prompt = diff
		}

		system := `You are a code reviewer applying the "Surgical Changes" principle.

Analyze the diff and evaluate:
1. **Surgical Score** (0-10): How surgical are the changes? 10 = only exactly what was needed, 0 = full of unrelated changes.
2. **Violations**: List every changed line that does NOT trace directly to the stated task:
   - Drive-by formatting or style fixes
   - Refactoring of unrelated code
   - Added features nobody asked for
   - Changed comments or docstrings unrelated to the fix
   - New type hints, imports, or variable names that weren't needed
3. **Verdict**: PASS (score ≥ 7) or FAIL (score < 7)

The test: "Does every changed line trace directly to the user's request?"`

		result, err := Ask(ctx, s, system, prompt)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	}
	return tool, handler
}
