package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Parcifallll/trip-go/internal/config"
	"github.com/Parcifallll/trip-go/internal/container"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := container.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build container: %v\n", err)
		return 1
	}

	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "run service: %v\n", err)
		return 1
	}

	return 0
}
