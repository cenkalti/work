package harness

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func goalTool(s *server.MCPServer) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("goal",
		mcp.WithDescription("Transform a vague task into a numbered plan with verifiable success criteria. Each step includes a '→ verify:' check. Favors test-first where applicable."),
		mcp.WithString("task",
			mcp.Required(),
			mcp.Description("The vague task or feature request to transform into verifiable goals"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		task, err := req.RequireString("task")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		prompt := fmt.Sprintf(`Transform this task into verifiable goals:

"%s"

Apply the "Goal-Driven Execution" principle:

1. Identify the **specific problem** being solved (not the general area).
2. Write a numbered **step-by-step plan** where each step has a concrete verification:
   1. [Step] → verify: [specific check]
   2. [Step] → verify: [specific check]
3. For bug fixes: start with "Write a test that reproduces the bug → verify: test fails"
4. For new features: start with "Write tests for the expected behavior → verify: tests fail (red)"
5. End with "Run full test suite → verify: all tests pass (green)"

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.`, task)

		result, err := Ask(ctx, s, "", prompt)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	}
	return tool, handler
}
