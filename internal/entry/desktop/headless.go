package desktop

import "pause/internal/app"

func RunHeadless(configPath string) error {
	return app.RunHeadless(configPath)
}
