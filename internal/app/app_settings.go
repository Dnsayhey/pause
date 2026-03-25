package app

import (
	"errors"

	"pause/internal/backend/domain/settings"
	"pause/internal/logx"
)

func (a *App) GetSettings() settings.Settings {
	if a == nil {
		return settings.DefaultSettings()
	}
	if a.settingsSvc != nil {
		return a.settingsSvc.Get(appContextOrBackground(a.ctx))
	}
	return settings.DefaultSettings()
}

func (a *App) UpdateSettings(patch settings.SettingsPatch) (settings.Settings, error) {
	if a == nil {
		return settings.Settings{}, errors.New("app unavailable")
	}
	if a.settingsSvc == nil {
		return settings.Settings{}, errors.New("settings service unavailable")
	}

	nextSettings, err := a.settingsSvc.Update(appContextOrBackground(a.ctx), patch)
	if err != nil {
		logx.Warnf("app.update_settings_err err=%v", err)
		return settings.Settings{}, err
	}
	return nextSettings, nil
}

func (a *App) GetLaunchAtLogin() (bool, error) {
	if a == nil {
		return false, errors.New("app unavailable")
	}
	if a.settingsSvc != nil {
		return a.settingsSvc.GetLaunchAtLogin(appContextOrBackground(a.ctx))
	}
	return false, errors.New("settings service unavailable")
}

func (a *App) SetLaunchAtLogin(enabled bool) (bool, error) {
	if a == nil {
		return false, errors.New("app unavailable")
	}
	if a.settingsSvc != nil {
		return a.settingsSvc.SetLaunchAtLogin(appContextOrBackground(a.ctx), enabled)
	}
	return false, errors.New("settings service unavailable")
}
