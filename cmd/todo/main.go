package main

import (
	"os"

	"github.com/cenkalti/work/internal/cli/todo"
)

func main() {
	if err := todo.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
