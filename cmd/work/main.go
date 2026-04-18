package main

import (
	"os"

	"github.com/cenkalti/work/internal/cli/work"
)

func main() {
	if err := work.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
