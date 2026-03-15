package cli

import (
	"os"

	mcpserver "github.com/cenkalti/work/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for task creation",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := workContext(cmd)
			goal, err := ctx.ResolveGoal("")
			if err != nil {
				return err
			}
			tasksDir := tasksDirFor(ctx.RootRepo, goal)
			s := mcpserver.NewServer(tasksDir)
			return server.NewStdioServer(s).Listen(cmd.Context(), os.Stdin, os.Stdout)
		},
	}
}
