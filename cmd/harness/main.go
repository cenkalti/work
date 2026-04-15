package main

import (
	"context"
	"log"
	"os"

	"github.com/cenkalti/work/internal/harness"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := harness.NewServer()
	if err := server.NewStdioServer(s).Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
