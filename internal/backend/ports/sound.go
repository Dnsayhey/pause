package ports

import "pause/internal/core/settings"

type SoundPlayer interface {
	PlayBreakEnd(sound settings.SoundSettings) error
}
