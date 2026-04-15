package harness

import "github.com/mark3labs/mcp-go/server"

func NewServer() *server.MCPServer {
	s := server.NewMCPServer("harness", "1.0.0")
	s.EnableSampling()
	s.AddTool(thinkTool(s))
	s.AddTool(simplifyTool(s))
	s.AddTool(surgicalTool(s))
	s.AddTool(goalTool(s))
	s.AddTool(scoreTool(s))
	s.AddTool(benchTool(s))
	return s
}
