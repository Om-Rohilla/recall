package main

import (
	"os"

	"github.com/Om-Rohilla/recall/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
