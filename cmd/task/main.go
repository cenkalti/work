package main

import (
	"os"

	"github.com/cenkalti/work/internal/cli/task"
)

func main() {
	if err := task.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
