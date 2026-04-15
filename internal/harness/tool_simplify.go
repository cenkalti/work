package harness

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func simplifyTool(s *server.MCPServer) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("simplify",
		mcp.WithDescription("Analyze code for over-engineering. Identifies unnecessary abstractions, speculative features, and excess complexity. Suggests the simplest version that meets the actual requirements."),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description("The code to analyze for unnecessary complexity"),
		),
		mcp.WithString("context",
			mcp.Description("Optional: what the code is supposed to do (the original task or requirement)"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		code, err := req.RequireString("code")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		taskContext := req.GetString("context", "")

		var prompt string
		if taskContext != "" {
			prompt = fmt.Sprintf("Original requirement: %s\n\nCode to review:\n%s", taskContext, code)
		} else {
			prompt = code
		}

		system := `You are a senior engineer applying the "Simplicity First" principle. Your job is to identify over-engineering.

For the given code, evaluate:
1. **Complexity Score** (0-10): How over-engineered is it? 0 = perfectly simple, 10 = massively over-engineered.
2. **Issues**: List every unnecessary abstraction, speculative feature, unneeded configurability, or impossible error handling.
3. **Simplified Version**: Rewrite the simplest version that solves the actual stated problem. If no context was given, assume the most minimal reasonable requirement.
4. **Line Count**: State the original line count vs. your simplified version.

The test: "Would a senior engineer say this is overcomplicated?" If yes, simplify it.`

		result, err := Ask(ctx, s, system, prompt)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	}
	return tool, handler
}
