//go:build wails

package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
	"pause/internal/desktop"
	"pause/internal/logx"
)

func overlaySkipMode(settings settings.Settings) skipMode {
	if settings.Enforcement.OverlaySkipAllowed {
		return skipModeNormal
	}
	return skipModeEmergency
}

func (c *wailsDesktopController) syncOverlay(ctx context.Context, state state.RuntimeState, settings settings.Settings) {
	overlayActive := state.CurrentSession != nil && state.CurrentSession.Status == "resting"
	overlaySkipAllowed := overlayActive && state.OverlaySkipAllowed && state.CurrentSession != nil && state.CurrentSession.CanSkip
	language := c.lastLanguage
	theme := resolveEffectiveTheme(settings.UI.Theme)
	overlayText := ""
	overlayMessage := overlayMessageText(language)
	if overlayActive && state.CurrentSession != nil {
		overlayText = overlayCountdownText(language, state.CurrentSession.RemainingSec)
		if !state.CurrentSession.StartedAt.Equal(c.lastOverlaySessionStart) {
			c.lastOverlaySessionStart = state.CurrentSession.StartedAt
			c.overlayFailureLogged = false
		}
	} else {
		c.lastOverlaySessionStart = time.Time{}
		c.overlayFailureLogged = false
	}

	if c.overlay.IsNative() {
		needsUpdate := overlayActive != c.lastOverlayActive || overlaySkipAllowed != c.lastOverlaySkip || language != c.lastOverlayLang || overlayText != c.lastOverlayText || theme != c.lastOverlayTheme
		if needsUpdate {
			if overlayActive {
				// Keep native break overlay isolated from the main window.
				desktop.HideMainWindowForOverlay(ctx)
				if !c.overlay.Show(overlaySkipAllowed, overlaySkipButtonTitle(language), overlayText, overlayMessage, theme) {
					if !c.overlayFailureLogged {
						logx.Warnf(
							"overlay.show_failed native=true reasons=%s remaining_sec=%d",
							overlayReasons(state),
							overlayRemainingSec(state),
						)
						c.overlayFailureLogged = true
					}
				}
			} else {
				c.overlay.Hide()
			}
		}
		c.lastOverlayActive = overlayActive
		c.lastOverlaySkip = overlaySkipAllowed
		c.lastOverlayLang = language
		c.lastOverlayText = overlayText
		c.lastOverlayTheme = theme
		return
	}

	if overlayActive && !c.overlayFailureLogged {
		logx.Warnf(
			"overlay.show_failed native=false reasons=%s remaining_sec=%d",
			overlayReasons(state),
			overlayRemainingSec(state),
		)
		c.overlayFailureLogged = true
	}
	c.lastOverlayActive = overlayActive
	c.lastOverlaySkip = overlaySkipAllowed
	c.lastOverlayLang = language
	c.lastOverlayText = overlayText
	c.lastOverlayTheme = theme
}

func overlayReasons(state state.RuntimeState) string {
	if state.CurrentSession == nil || len(state.CurrentSession.Reasons) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(state.CurrentSession.Reasons))
	for _, reason := range state.CurrentSession.Reasons {
		parts = append(parts, fmt.Sprintf("%d", reason))
	}
	return strings.Join(parts, "+")
}

func overlayRemainingSec(state state.RuntimeState) int {
	if state.CurrentSession == nil {
		return 0
	}
	return state.CurrentSession.RemainingSec
}
