//go:build !wails

package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	app, err := NewApp("")
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app.Startup(ctx)
	<-ctx.Done()
}
