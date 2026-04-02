package app

import "runtime"

func (a *App) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		OS:   normalizePlatformOS(runtime.GOOS),
		Arch: normalizePlatformArch(runtime.GOARCH),
	}
}

func normalizePlatformOS(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return "unknown"
	}
}

func normalizePlatformArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return "unknown"
	}
}
