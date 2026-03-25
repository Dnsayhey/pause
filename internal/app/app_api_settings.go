package app

import (
	"errors"

	settingsdomain "pause/internal/backend/domain/settings"
	"pause/internal/logx"
)

func (a *App) GetSettings() Settings {
	if a == nil {
		return settingsFromDomain(settingsdomain.DefaultSettings())
	}
	if a.settingsSvc != nil {
		return settingsFromDomain(a.settingsSvc.Get(appContextOrBackground(a.ctx)))
	}
	return settingsFromDomain(settingsdomain.DefaultSettings())
}

func (a *App) UpdateSettings(patch SettingsPatch) (Settings, error) {
	if a == nil {
		return Settings{}, errors.New("app unavailable")
	}
	if a.settingsSvc == nil {
		return Settings{}, errors.New("settings service unavailable")
	}

	nextSettings, err := a.settingsSvc.Update(appContextOrBackground(a.ctx), settingsPatchToDomain(patch))
	if err != nil {
		logx.Warnf("app.update_settings_err err=%v", err)
		return Settings{}, err
	}
	return settingsFromDomain(nextSettings), nil
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
