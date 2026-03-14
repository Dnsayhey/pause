package app

import (
	"context"
	"os/signal"
	"syscall"
)

func RunHeadless(configPath string) error {
	desktopApp, err := NewApp(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	desktopApp.Startup(ctx)
	<-ctx.Done()
	return nil
}
