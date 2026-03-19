package main

import (
	"os"

	"github.com/cenkalti/work/internal/cli/work"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := work.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
