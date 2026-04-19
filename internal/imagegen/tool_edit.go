package imagegen

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func editImageTool(cfg Config, c *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("edit_image",
		mcp.WithDescription("Edit an existing image (inpainting or variation) using OpenAI gpt-image-1 and write the result to disk."),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("Natural-language description of the desired edit."),
		),
		mcp.WithString("input_path",
			mcp.Required(),
			mcp.Description("Absolute path to the existing image on disk."),
		),
		mcp.WithString("mask_path",
			mcp.Description("Optional absolute path to a PNG mask. The alpha channel marks the editable region."),
		),
		mcp.WithString("output_path",
			mcp.Required(),
			mcp.Description("Absolute path to write the edited image. Extension must match format. For n>1, a -1, -2, ... suffix is inserted before the extension."),
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
		inputPath, err := req.RequireString("input_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		outputPath, err := req.RequireString("output_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		er := EditRequest{
			GenerateRequest: GenerateRequest{
				Prompt:       prompt,
				Size:         req.GetString("size", "1024x1024"),
				Background:   req.GetString("background", "auto"),
				OutputFormat: req.GetString("format", "png"),
				Quality:      req.GetString("quality", "high"),
				N:            req.GetInt("n", 1),
			},
			ImagePath: inputPath,
			MaskPath:  req.GetString("mask_path", ""),
		}

		resolved, err := ResolveOutputPath(outputPath, cfg.DefaultDir)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := ValidateEdit(er, resolved); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := runImageJob(ctx, resolved, er.N, func() (*ImageResponse, error) {
			return c.Edit(ctx, er)
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructured(result, summarize(result)), nil
	}

	return tool, handler
}
