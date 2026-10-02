package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"persistty/internal/elevation"
)

func main() {
	if len(os.Args) != 1 {
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer stop()
	if elevation.RunHelper(ctx) != nil {
		os.Exit(1)
	}
}
