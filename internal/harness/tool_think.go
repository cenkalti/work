package harness

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func thinkTool(s *server.MCPServer) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("think",
		mcp.WithDescription("Before writing any code, surface all assumptions, ambiguities, and tradeoffs for a task. Returns a structured analysis to guide implementation."),
		mcp.WithString("task",
			mcp.Required(),
			mcp.Description("The task or feature request to analyze before coding"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		task, err := req.RequireString("task")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		prompt := fmt.Sprintf(`A developer asked you to implement the following task:

"%s"

Before writing any code, apply the "Think Before Coding" principle:

1. **Assumptions** — List every assumption you are making. If you are uncertain about any of them, flag it.
2. **Interpretations** — If the request has multiple reasonable interpretations, list them all. Do not silently pick one.
3. **Tradeoffs** — For each interpretation or approach, state the key tradeoffs.
4. **Clarifying Questions** — List the questions you would ask the developer before starting, in priority order.
5. **Simplest Approach** — Briefly describe the simplest possible implementation if all assumptions hold.

Do not write any code. Do not start implementing. Just think out loud.`, task)

		result, err := Ask(ctx, s, "", prompt)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	}
	return tool, handler
}
