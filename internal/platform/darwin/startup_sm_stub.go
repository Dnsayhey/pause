//go:build darwin && !cgo

package darwin

type smMode int

const (
	smModeUnavailable smMode = 0
	smModeMainApp     smMode = 1
	smModeLoginItem   smMode = 2
)

func smStartupMode() smMode {
	return smModeUnavailable
}

func smSetLaunchAtLogin(_ string, _ bool) error {
	return errSMUnsupported
}

func smGetLaunchAtLogin(_ string) (bool, error) {
	return false, errSMUnsupported
}
