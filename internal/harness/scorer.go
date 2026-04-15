package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Score struct {
	Principle string `json:"principle"`
	Value     int    `json:"value"`
	Reasoning string `json:"reasoning"`
}

type judgeResult struct {
	Score     int    `json:"score"`
	Reasoning string `json:"reasoning"`
}

// Ask sends a prompt to the client via MCP sampling. The client (Claude Code)
// chooses the model.
func Ask(ctx context.Context, s *server.MCPServer, system, userPrompt string) (string, error) {
	req := mcp.CreateMessageRequest{}
	req.MaxTokens = 2048
	req.SystemPrompt = system
	req.Messages = []mcp.SamplingMessage{
		{
			Role:    mcp.RoleUser,
			Content: mcp.TextContent{Type: "text", Text: userPrompt},
		},
	}

	result, err := s.RequestSampling(ctx, req)
	if err != nil {
		return "", err
	}
	return extractSamplingText(result.Content)
}

// ScoreResponse asks the client to judge a response against each principle.
func ScoreResponse(ctx context.Context, s *server.MCPServer, response string, principles []Principle) ([]Score, error) {
	var scores []Score
	for _, p := range principles {
		prompt, err := renderJudgePrompt(p.JudgePrompt, response)
		if err != nil {
			return nil, fmt.Errorf("principle %s: render prompt: %w", p.ID, err)
		}

		req := mcp.CreateMessageRequest{}
		req.MaxTokens = 256
		req.Messages = []mcp.SamplingMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.TextContent{Type: "text", Text: prompt},
			},
		}

		result, err := s.RequestSampling(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("principle %s: sampling: %w", p.ID, err)
		}

		raw, err := extractSamplingText(result.Content)
		if err != nil {
			return nil, fmt.Errorf("principle %s: extract text: %w", p.ID, err)
		}

		var jr judgeResult
		if err := json.Unmarshal([]byte(raw), &jr); err != nil {
			return nil, fmt.Errorf("principle %s: parse judge response %q: %w", p.ID, raw, err)
		}

		scores = append(scores, Score{
			Principle: p.Name,
			Value:     jr.Score,
			Reasoning: jr.Reasoning,
		})
	}
	return scores, nil
}

func renderJudgePrompt(tmpl, response string) (string, error) {
	t, err := template.New("judge").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, struct{ Response string }{response}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func extractSamplingText(content any) (string, error) {
	switch v := content.(type) {
	case mcp.TextContent:
		return v.Text, nil
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			return text, nil
		}
	}
	return "", fmt.Errorf("unexpected sampling content type %T", content)
}
