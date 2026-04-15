package main

import (
	"os"

	"github.com/cenkalti/work/internal/cli/agent"
)

func main() {
	if err := agent.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
