package main

import (
	"fmt"
	"os"

	"github.com/ixxet/hermes/internal/command"
)

var version = "dev"

func main() {
	if err := command.Execute(os.Args[1:], command.Dependencies{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: version,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
