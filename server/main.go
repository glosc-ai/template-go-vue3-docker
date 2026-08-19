package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// The distroless production image has no shell, wget, or curl, so a
	// standard Docker HEALTHCHECK can't exec a probe command. Running this
	// binary with -healthcheck instead makes an HTTP request to its own
	// /health/live and exits 0/1, which HEALTHCHECK can invoke directly.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runHealthcheck())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("application stopped", "err", err)
		os.Exit(1)
	}
}
