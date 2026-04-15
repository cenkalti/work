package harness

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func scoreTool(s *server.MCPServer) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("score",
		mcp.WithDescription("Score an LLM response against all four Karpathy principles (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution). Returns a 0-10 score with reasoning for each principle."),
		mcp.WithString("response",
			mcp.Required(),
			mcp.Description("The LLM response text to evaluate"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		response, err := req.RequireString("response")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		scores, err := ScoreResponse(ctx, s, response, All)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var sb strings.Builder
		sb.WriteString("Karpathy Principles Score\n")
		sb.WriteString("=========================\n\n")
		for _, s := range scores {
			sb.WriteString(fmt.Sprintf("**%s**: %d/10\n%s\n\n", s.Principle, s.Value, s.Reasoning))
		}
		return mcp.NewToolResultText(sb.String()), nil
	}
	return tool, handler
}
