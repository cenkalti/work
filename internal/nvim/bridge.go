package nvim

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewServer(socket string) *server.MCPServer {
	s := server.NewMCPServer("nvim-bridge", "1.0.0")
	s.AddTool(execLuaTool(socket))
	return s
}

func execLuaTool(socket string) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("nvim_exec_lua",
		mcp.WithDescription(
			"Execute a Lua chunk inside the connected NeoVim instance and return the result.\n\n"+
				"WHEN TO CALL THIS TOOL:\n"+
				"- Before answering any question about the code: call it first to fetch the "+
				"current buffer contents, cursor position, diagnostics, open files, or any "+
				"other editor state you need.\n"+
				"- Whenever you produce Lua intended to mutate NeoVim (insert or replace text, "+
				"open files, show notifications, move the cursor, etc.): execute it here "+
				"instead of just printing it.\n\n"+
				"RETURNING DATA:\n"+
				"Use `return` in your Lua to pass values back. Tables are serialised with "+
				"vim.inspect() and returned as a string.\n\n"+
				"FULL API AVAILABLE:\n"+
				"vim.api, vim.fn, vim.lsp, vim.diagnostic, vim.bo, vim.wo, etc. "+
				"The working directory is the project root.",
		),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description(
				"Lua chunk to execute. May span multiple lines and contain local variables, "+
					"loops, function definitions, etc. Use `return expr` to get a value back.",
			),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		code, err := req.RequireString("code")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := execLua(socket, code)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

// execLua writes code to a temp file, then has NeoVim execute it via its RPC
// socket. The Lua wrapper captures the return value (or error) and writes it
// to a second temp file so we can read it back without going through
// Vimscript's lossy type serialisation.
func execLua(socket, code string) (string, error) {
	luaFile, err := os.CreateTemp("", "nvim_mcp_*.lua")
	if err != nil {
		return "", fmt.Errorf("create temp lua file: %w", err)
	}
	luaPath := luaFile.Name()
	resultPath := luaPath + ".result"

	wrapper := fmt.Sprintf(`
local _ok, _val = pcall(function()
%s
end)
local f = assert(io.open(%q, 'w'))
if not _ok then
  f:write('ERROR: ' .. tostring(_val))
elseif _val ~= nil then
  f:write(type(_val) == 'string' and _val or vim.inspect(_val))
end
f:close()
`, code, resultPath)

	if _, err := luaFile.WriteString(wrapper); err != nil {
		luaFile.Close()
		os.Remove(luaPath)
		return "", fmt.Errorf("write lua file: %w", err)
	}
	luaFile.Close()

	defer func() {
		os.Remove(luaPath)
		os.Remove(resultPath)
	}()

	out, err := exec.Command(
		"nvim", "--server", socket,
		"--remote-expr", fmt.Sprintf("luaeval(\"dofile('%s')\")", luaPath),
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("nvim RPC: %w\n%s", err, out)
	}

	result, err := os.ReadFile(resultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "(no return value)", nil
		}
		return "", fmt.Errorf("read result: %w", err)
	}

	return string(result), nil
}
