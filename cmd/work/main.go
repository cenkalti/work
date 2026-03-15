package main

import (
	"os"

	"github.com/cenkalti/work/internal/cli"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := cli.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
