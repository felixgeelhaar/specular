package main

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/specular/internal/cmd"
	"github.com/felixgeelhaar/specular/internal/exitcode"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitcode.ExitWithError(err)
	}
	exitcode.Exit(exitcode.Success)
}
