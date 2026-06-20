package main

import (
	"os"

	"github.com/hrodrig/kui/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
