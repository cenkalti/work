package harness

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func benchTool(s *server.MCPServer) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bench",
		mcp.WithDescription("Run the built-in benchmark suite (8 cases from the Karpathy examples). Tests Claude with and without the guidelines, then reports per-principle score improvement."),
		mcp.WithString("guidelines",
			mcp.Description("Path to the CLAUDE.md guidelines file. Defaults to CLAUDE.md in the current directory."),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		guidelinesPath := req.GetString("guidelines", "CLAUDE.md")

		guidelinesContent, err := readFile(guidelinesPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("read guidelines %s: %v", guidelinesPath, err)), nil
		}

		type caseResult struct {
			principleID     string
			baselineScore   int
			guidelinesScore int
		}

		var (
			mu      sync.Mutex
			results []caseResult
			sem     = make(chan struct{}, 3)
		)

		var wg sync.WaitGroup
		errCh := make(chan error, len(Builtin))

		for _, c := range Builtin {
			wg.Add(1)
			go func(c Case) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				userPrompt := c.Prompt
				if c.Context != "" {
					userPrompt = fmt.Sprintf("%s\n\nExisting code:\n```\n%s\n```", c.Prompt, c.Context)
				}

				baseline, err := Ask(ctx, s, "", userPrompt)
				if err != nil {
					errCh <- fmt.Errorf("case %s baseline: %w", c.ID, err)
					return
				}
				withGuidelines, err := Ask(ctx, s, guidelinesContent, userPrompt)
				if err != nil {
					errCh <- fmt.Errorf("case %s with-guidelines: %w", c.ID, err)
					return
				}

				for _, pid := range c.Principles {
					p := principleByID(pid)
					if p == nil {
						continue
					}
					bScores, err := ScoreResponse(ctx, s, baseline, []Principle{*p})
					if err != nil {
						errCh <- fmt.Errorf("case %s score baseline: %w", c.ID, err)
						return
					}
					gScores, err := ScoreResponse(ctx, s, withGuidelines, []Principle{*p})
					if err != nil {
						errCh <- fmt.Errorf("case %s score guidelines: %w", c.ID, err)
						return
					}
					mu.Lock()
					results = append(results, caseResult{
						principleID:     pid,
						baselineScore:   bScores[0].Value,
						guidelinesScore: gScores[0].Value,
					})
					mu.Unlock()
				}
			}(c)
		}
		wg.Wait()
		close(errCh)

		if err := <-errCh; err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		type agg struct {
			name                   string
			baselineSum            int
			guidelinesSum          int
			wins, ties, losses, count int
		}
		aggMap := map[string]*agg{}
		for _, p := range All {
			aggMap[p.ID] = &agg{name: p.Name}
		}
		for _, r := range results {
			a := aggMap[r.principleID]
			if a == nil {
				continue
			}
			a.baselineSum += r.baselineScore
			a.guidelinesSum += r.guidelinesScore
			a.count++
			switch {
			case r.guidelinesScore > r.baselineScore:
				a.wins++
			case r.guidelinesScore == r.baselineScore:
				a.ties++
			default:
				a.losses++
			}
		}

		var sb strings.Builder
		sb.WriteString("Benchmark Results\n")
		sb.WriteString("=================\n\n")
		fmt.Fprintf(&sb, "%-25s  %8s  %15s  %6s  %9s\n", "PRINCIPLE", "BASELINE", "WITH GUIDELINES", "DELTA", "W/T/L")
		sb.WriteString(strings.Repeat("-", 70) + "\n")
		for _, p := range All {
			a := aggMap[p.ID]
			if a == nil || a.count == 0 {
				continue
			}
			base := float64(a.baselineSum) / float64(a.count)
			guide := float64(a.guidelinesSum) / float64(a.count)
			delta := guide - base
			fmt.Fprintf(&sb, "%-25s  %8.1f  %15.1f  %+6.1f  %d/%d/%d\n",
				a.name, base, guide, delta, a.wins, a.ties, a.losses)
		}
		return mcp.NewToolResultText(sb.String()), nil
	}
	return tool, handler
}

func principleByID(id string) *Principle {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}
