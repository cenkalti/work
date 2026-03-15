package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/cenkalti/work/internal/task"
)

var validTaskID = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var taskSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"properties": map[string]any{
		"tasks": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"id":          map[string]any{"type": "string", "description": "Short kebab-case slug for the task"},
					"task":        map[string]any{"type": "string", "description": "One-line summary of what to do"},
					"depends_on":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Array of task ids this depends on"},
					"files":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "File paths this task touches"},
					"description": map[string]any{"type": "string", "description": "Detailed description of what to do and how"},
					"acceptance":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "List of acceptance criteria"},
					"context":     map[string]any{"type": "string", "description": "Additional context: links, notes, references"},
				},
				"required": []string{"id", "task", "depends_on", "files", "description", "acceptance", "context"},
			},
		},
	},
	"required": []string{"tasks"},
}

type tasksResponse struct {
	Tasks []task.Task `json:"tasks"`
}

func ExtractTasks(ctx context.Context, planContent, instructions string) ([]task.Task, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set, get your API key from https://console.anthropic.com/settings/keys and set it with: export ANTHROPIC_API_KEY=sk-ant-xxx")
	}

	client := anthropic.NewClient()

	prompt := "Extract every task from this plan as structured JSON. Each task should be a separate item. Parse the dependency graph carefully — use the task ids you assign to reference dependencies. Break down the work into small, focused, implementable tasks."
	if instructions != "" {
		prompt += "\n\nADDITIONAL INSTRUCTIONS:\n" + instructions
	}
	prompt += "\n\nPLAN:\n" + planContent

	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-opus-4-6",
		MaxTokens: 16384,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(prompt),
			),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: taskSchema,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api call failed: %w", err)
	}

	if message.StopReason != "end_turn" {
		fmt.Fprintf(os.Stderr, "Warning: stop_reason=%s (expected end_turn)\n", message.StopReason)
	}

	if len(message.Content) == 0 {
		return nil, fmt.Errorf("empty response from claude")
	}

	var text string
	for _, block := range message.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	if text == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	var resp tasksResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		fmt.Fprintf(os.Stderr, "Raw response:\n%s\n", text)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Tasks) == 0 {
		fmt.Fprintf(os.Stderr, "Raw response:\n%s\n", text)
		return nil, fmt.Errorf("no tasks extracted")
	}

	var invalidIDs []string
	for i := range resp.Tasks {
		if !validTaskID.MatchString(resp.Tasks[i].ID) {
			invalidIDs = append(invalidIDs, resp.Tasks[i].ID)
		}
		if resp.Tasks[i].Status == "" {
			resp.Tasks[i].Status = task.StatusPending
		}
	}
	if len(invalidIDs) > 0 {
		return nil, fmt.Errorf("invalid task IDs (must be kebab-case): %v", invalidIDs)
	}

	return resp.Tasks, nil
}
