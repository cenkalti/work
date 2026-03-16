package cli

import (
	"os"

	mcpserver "github.com/cenkalti/work/internal/mcp"
	"github.com/cenkalti/work/internal/paths"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for task creation",
		RunE: func(cmd *cobra.Command, args []string) error {
			loc := detectLocation(cmd)
			branch, err := loc.ResolveBranch("")
			if err != nil {
				return err
			}
			s := mcpserver.NewServer(paths.TasksDir(loc.RootRepo, branch))
			return server.NewStdioServer(s).Listen(cmd.Context(), os.Stdin, os.Stdout)
		},
	}
}
