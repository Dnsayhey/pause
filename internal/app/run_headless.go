package app

import (
	"context"
	"os/signal"
	"syscall"

	"pause/internal/logx"
)

func RunHeadless(configPath string) error {
	desktopApp, err := NewApp(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	desktopApp.Startup(ctx)
	logx.Infof("app.headless_running")
	<-ctx.Done()
	logx.Infof("app.headless_stopped")
	return nil
}
