package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/nvim"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	socket := flag.String("socket", "", "NeoVim RPC socket path (vim.v.servername)")
	flag.Parse()

	if *socket == "" {
		fmt.Fprintln(os.Stderr, "nvim-bridge: -socket flag is required")
		os.Exit(1)
	}

	s := nvim.NewServer(*socket)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, "nvim-bridge:", err)
		os.Exit(1)
	}
}
