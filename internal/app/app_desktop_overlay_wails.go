//go:build wails

package app

import (
	"context"
	"time"

	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/state"
	"pause/internal/desktop"
)

func overlaySkipMode(settings settings.Settings) skipMode {
	if settings.Enforcement.OverlaySkipAllowed {
		return skipModeNormal
	}
	return skipModeEmergency
}

func (c *wailsDesktopController) syncOverlay(ctx context.Context, app *App, state state.RuntimeState, settings settings.Settings) {
	overlayActive := state.CurrentSession != nil && state.CurrentSession.Status == "resting"
	overlaySkipAllowed := overlayActive && state.OverlaySkipAllowed && state.CurrentSession != nil && state.CurrentSession.CanSkip
	language := c.lastLanguage
	theme := resolveEffectiveTheme(settings.UI.Theme)
	overlayText := ""
	if overlayActive && state.CurrentSession != nil {
		overlayText = overlayCountdownText(language, state.CurrentSession.RemainingSec)
		if !state.CurrentSession.StartedAt.Equal(c.lastOverlaySessionStart) {
			c.lastOverlaySessionStart = state.CurrentSession.StartedAt
			c.overlayFallbackNotified = false
		}
	} else {
		c.lastOverlaySessionStart = time.Time{}
		c.overlayFallbackNotified = false
	}

	if c.overlay.IsNative() {
		needsUpdate := overlayActive != c.lastOverlayActive || overlaySkipAllowed != c.lastOverlaySkip || language != c.lastOverlayLang || overlayText != c.lastOverlayText || theme != c.lastOverlayTheme
		if needsUpdate {
			if overlayActive {
				// Keep native break overlay isolated from the main window.
				desktop.HideMainWindowForOverlay(ctx)
				if !c.overlay.Show(overlaySkipAllowed, overlaySkipButtonTitle(language), overlayText, theme) {
					if !c.overlayFallbackNotified && app != nil {
						app.sendBreakFallbackNotification(state)
						c.overlayFallbackNotified = true
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

	if overlayActive && !c.overlayFallbackNotified && app != nil {
		app.sendBreakFallbackNotification(state)
		c.overlayFallbackNotified = true
	}
	c.lastOverlayActive = overlayActive
	c.lastOverlaySkip = overlaySkipAllowed
	c.lastOverlayLang = language
	c.lastOverlayText = overlayText
	c.lastOverlayTheme = theme
}
