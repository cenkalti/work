package mcpserver

import (
	"context"
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/task"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewServer(tasksDir string) *server.MCPServer {
	s := server.NewMCPServer("work", "1.0.0")
	s.AddTool(createTaskTool(tasksDir))
	return s
}

func createTaskTool(tasksDir string) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("create_task",
		mcp.WithDescription("Create a task JSON file in the tasks directory"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Task ID in kebab-case (e.g. add-mcp-server)"),
		),
		mcp.WithString("task",
			mcp.Required(),
			mcp.Description("One-line task summary"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Detailed description of the task"),
		),
		mcp.WithArray("acceptance",
			mcp.Required(),
			mcp.Description("List of acceptance criteria strings"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("depends_on",
			mcp.Description("List of task IDs this task depends on"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("files",
			mcp.Description("List of files relevant to this task"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("context",
			mcp.Description("Additional context or notes"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary, err := req.RequireString("task")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		description, err := req.RequireString("description")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		acceptance, err := req.RequireStringSlice("acceptance")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dependsOn := req.GetStringSlice("depends_on", nil)
		files := req.GetStringSlice("files", nil)
		taskContext := req.GetString("context", "")

		t := &task.Task{
			ID:          id,
			Summary: summary,
			DependsOn:   dependsOn,
			Status:      task.StatusPending,
			Files:       files,
			Description: description,
			Acceptance:  acceptance,
			Context:     taskContext,
		}

		existing, err := task.LoadAll(tasksDir)
		if err != nil && !os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("loading tasks: %v", err)), nil
		}
		if existing == nil {
			existing = make(map[string]*task.Task)
		}
		existing[t.ID] = t
		if err := task.DetectCycle(existing); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if err := t.WriteToFile(tasksDir); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("writing task: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("created: %s", id)), nil
	}

	return tool, handler
}
