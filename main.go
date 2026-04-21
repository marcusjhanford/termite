package main

import (
	"os"

	"github.com/termite-mail/termite/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
