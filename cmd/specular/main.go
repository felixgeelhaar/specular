package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/specular/internal/cmd"
	"github.com/felixgeelhaar/specular/internal/exitcode"
)

func main() {
	// Create a context that listens for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.ExecuteContext(ctx); err != nil {
		// Check if error was due to context cancellation (e.g., Ctrl+C)
		if ctx.Err() == context.Canceled {
			fmt.Fprintln(os.Stderr, "\nOperation cancelled by user")
			exitcode.Exit(exitcode.Interrupted)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitcode.ExitWithError(err)
	}
	exitcode.Exit(exitcode.Success)
}
