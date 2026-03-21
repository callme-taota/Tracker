package main

import (
	"Tracker/internal/cli"
	"os"
)

func main() {
	if err := cli.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
