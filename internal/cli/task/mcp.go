package task

import (
	"os"

	mcpserver "github.com/cenkalti/work/internal/mcp"
	"github.com/cenkalti/work/internal/paths"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "mcp",
		Short:  "Start MCP server for task creation",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			s := mcpserver.NewServer(paths.LocalTasksDir(cwd))
			return server.NewStdioServer(s).Listen(cmd.Context(), os.Stdin, os.Stdout)
		},
	}
}
