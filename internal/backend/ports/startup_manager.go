package ports

type StartupManager interface {
	SetLaunchAtLogin(enabled bool) error
	GetLaunchAtLogin() (bool, error)
}
