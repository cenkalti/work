package imagegen

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

func NewServer() *server.MCPServer {
	cfg, err := Load()
	if err != nil {
		log.SetFlags(0)
		log.SetOutput(os.Stderr)
		log.Fatalf("image-gen: %v", err)
	}
	client := &Client{
		APIKey: cfg.APIKey,
		HTTP:   &http.Client{Timeout: 2 * time.Minute},
	}
	s := server.NewMCPServer("image-gen", "1.0.0")
	s.AddTool(generateImageTool(cfg, client))
	s.AddTool(editImageTool(cfg, client))
	return s
}
