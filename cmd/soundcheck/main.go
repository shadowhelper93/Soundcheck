// Package main provides the entry point for the Soundcheck CLI.
package main

import (
	"os"

	"github.com/soundcheck/soundcheck/internal/transport/cli"
)

func main() {
	os.Exit(cli.Execute())
}
