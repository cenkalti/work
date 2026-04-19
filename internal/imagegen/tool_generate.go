package imagegen

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func generateImageTool(cfg Config, c *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("generate_image",
		mcp.WithDescription("Generate one or more images from a text prompt using OpenAI gpt-image-1 and write them to disk."),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("Natural-language description of the image to generate."),
		),
		mcp.WithString("output_path",
			mcp.Required(),
			mcp.Description("Absolute path to write the image. Extension must match format. For n>1, a -1, -2, ... suffix is inserted before the extension."),
		),
		mcp.WithString("format",
			mcp.Description("Output image format."),
			mcp.Enum("png", "jpeg", "webp"),
			mcp.DefaultString("png"),
		),
		mcp.WithString("size",
			mcp.Description("Image size. gpt-image-1 accepts only these values."),
			mcp.Enum("1024x1024", "1536x1024", "1024x1536", "auto"),
			mcp.DefaultString("1024x1024"),
		),
		mcp.WithString("background",
			mcp.Description("Background mode. transparent is honored only for png and webp."),
			mcp.Enum("transparent", "opaque", "auto"),
			mcp.DefaultString("auto"),
		),
		mcp.WithString("quality",
			mcp.Description("Render quality. Transparency requires medium or high."),
			mcp.Enum("low", "medium", "high", "auto"),
			mcp.DefaultString("high"),
		),
		mcp.WithNumber("n",
			mcp.Description("Number of images to generate (1-10)."),
			mcp.DefaultNumber(1),
			mcp.Min(1),
			mcp.Max(10),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt, err := req.RequireString("prompt")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		outputPath, err := req.RequireString("output_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		gr := GenerateRequest{
			Prompt:       prompt,
			Size:         req.GetString("size", "1024x1024"),
			Background:   req.GetString("background", "auto"),
			OutputFormat: req.GetString("format", "png"),
			Quality:      req.GetString("quality", "high"),
			N:            req.GetInt("n", 1),
		}

		resolved, err := ResolveOutputPath(outputPath, cfg.DefaultDir)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := ValidateGenerate(gr, resolved); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := runImageJob(ctx, resolved, gr.N, func() (*ImageResponse, error) {
			return c.Generate(ctx, gr)
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructured(result, summarize(result)), nil
	}

	return tool, handler
}

func runImageJob(_ context.Context, resolved string, n int, call func() (*ImageResponse, error)) (*ToolResult, error) {
	paths := SuffixedPaths(resolved, n)
	for _, p := range paths {
		if err := ensureParentDir(p); err != nil {
			return nil, fmt.Errorf("create parent dir: %w", err)
		}
	}
	resp, err := call()
	if err != nil {
		return nil, err
	}
	if len(resp.Data) != len(paths) {
		return nil, fmt.Errorf("openai returned %d images, expected %d", len(resp.Data), len(paths))
	}
	files := make([]FileOut, 0, len(paths))
	for i, item := range resp.Data {
		bytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decode image %d: %w", i, err)
		}
		if err := os.WriteFile(paths[i], bytes, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", paths[i], err)
		}
		info, err := os.Stat(paths[i])
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", paths[i], err)
		}
		files = append(files, FileOut{Path: paths[i], SizeBytes: info.Size()})
	}
	return &ToolResult{
		Success: true,
		Files:   files,
		Model:   model,
		Usage:   resp.Usage,
	}, nil
}

func summarize(r *ToolResult) string {
	if len(r.Files) == 1 {
		return fmt.Sprintf("wrote %s (%d bytes)", r.Files[0].Path, r.Files[0].SizeBytes)
	}
	return fmt.Sprintf("wrote %d files", len(r.Files))
}
