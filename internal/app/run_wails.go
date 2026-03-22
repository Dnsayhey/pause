//go:build wails

package app

import (
	"io/fs"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"pause/internal/logx"
	"pause/internal/meta"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func InstallProcessSignalQuit(desktopApp *App) {
	if desktopApp == nil {
		return
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		logx.Warnf("app.signal_quit signal=interrupt_or_sigterm")
		desktopApp.Quit()
		signal.Stop(ch)
		close(ch)
	}()
}

func RunWails(configPath string, assets fs.FS) error {
	desktopApp, err := NewApp(configPath)
	if err != nil {
		return err
	}
	InstallProcessSignalQuit(desktopApp)

	return wails.Run(&options.App{
		Title:       "Pause",
		Width:       820,
		Height:      520,
		MinWidth:    820,
		MinHeight:   520,
		// Keep native title bars on macOS/Linux, but use frameless window on Windows.
		Frameless:   runtime.GOOS == "windows",
		StartHidden: true,
		// Keep this false and control close behavior in OnBeforeClose.
		// Wails' native HideWindowOnClose hides the whole app on macOS,
		// which can cause unexpected main-window re-show on status-item tooltip/activation flows.
		HideWindowOnClose: false,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
		},
		Windows: &windows.Options{
			// In frameless mode, keep system shadows/rounded corners when available.
			DisableFramelessWindowDecorations: false,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: meta.SingleInstanceID(),
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				logx.Infof(
					"app.second_instance_detected args=%d workdir=%s",
					len(secondInstanceData.Args),
					secondInstanceData.WorkingDirectory,
				)
			},
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     desktopApp.Startup,
		OnShutdown:    desktopApp.Shutdown,
		OnBeforeClose: desktopApp.BeforeClose,
		Bind: []interface{}{
			desktopApp,
		},
	})
}
