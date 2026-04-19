package imagegen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

const (
	model         = "gpt-image-1"
	apiGenerate   = "https://api.openai.com/v1/images/generations"
	apiEdit       = "https://api.openai.com/v1/images/edits"
	errBodySnippet = 512
)

type Client struct {
	APIKey string
	HTTP   *http.Client
}

func (c *Client) Generate(ctx context.Context, req GenerateRequest) (*ImageResponse, error) {
	body, err := json.Marshal(map[string]any{
		"model":         model,
		"prompt":        req.Prompt,
		"size":          req.Size,
		"background":    req.Background,
		"output_format": req.OutputFormat,
		"quality":       req.Quality,
		"n":             req.N,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiGenerate, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	return c.do(httpReq)
}

func (c *Client) Edit(ctx context.Context, req EditRequest) (*ImageResponse, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	writeErr := make(chan error, 1)

	go func() {
		defer pw.Close()
		defer mw.Close()
		writeErr <- buildEditMultipart(mw, req)
	}()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEdit, pr)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.do(httpReq)
	if werr := <-writeErr; werr != nil {
		return nil, fmt.Errorf("building multipart body: %w", werr)
	}
	return resp, err
}

func buildEditMultipart(mw *multipart.Writer, req EditRequest) error {
	fields := map[string]string{
		"model":         model,
		"prompt":        req.Prompt,
		"size":          req.Size,
		"background":    req.Background,
		"output_format": req.OutputFormat,
		"quality":       req.Quality,
		"n":             strconv.Itoa(req.N),
	}
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return err
		}
	}
	if err := attachFile(mw, "image", req.ImagePath); err != nil {
		return err
	}
	if req.MaskPath != "" {
		if err := attachFile(mw, "mask", req.MaskPath); err != nil {
			return err
		}
	}
	return nil
}

func attachFile(mw *multipart.Writer, field, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	w, err := mw.CreateFormFile(field, path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy %s: %w", path, err)
	}
	return nil
}

func (c *Client) do(req *http.Request) (*ImageResponse, error) {
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, errBodySnippet))
		return nil, fmt.Errorf("openai %s: %d: %s", req.URL.Path, resp.StatusCode, string(snippet))
	}

	var out ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
