//go:build wails

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func installProcessSignalQuit(app *App) {
	if app == nil {
		return
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		app.Quit()
		signal.Stop(ch)
		close(ch)
	}()
}

