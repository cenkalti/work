package imagegen

type GenerateRequest struct {
	Prompt       string
	Size         string
	Background   string
	OutputFormat string
	Quality      string
	N            int
}

type EditRequest struct {
	GenerateRequest
	ImagePath string
	MaskPath  string
}

type ImageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ImageResponse struct {
	Data  []ImageData `json:"data"`
	Usage ImageUsage  `json:"usage"`
}

type ImageData struct {
	B64JSON string `json:"b64_json"`
}

type FileOut struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
}

type ToolResult struct {
	Success bool       `json:"success"`
	Files   []FileOut  `json:"files"`
	Model   string     `json:"model"`
	Usage   ImageUsage `json:"usage"`
}
