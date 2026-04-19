package main

import (
	"context"
	"log"
	"os"

	"github.com/cenkalti/work/internal/imagegen"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := imagegen.NewServer()
	if err := server.NewStdioServer(s).Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
