package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appDirName    = "Pause"
	legacyDirName = ".pause"
	logsDirName   = "logs"
)

var (
	resolveUserConfigDir = os.UserConfigDir
	resolveUserCacheDir  = os.UserCacheDir
	resolveUserHomeDir   = os.UserHomeDir
	resolveGOOS          = func() string { return runtime.GOOS }
)

func ConfigDir() (string, error) {
	return appDataDir(resolveUserConfigDir)
}

func CacheDir() (string, error) {
	return appDataDir(resolveUserCacheDir)
}

func ConfigFile(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("config file name is required")
	}

	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func LogsDir() (string, error) {
	if resolveGOOS() == "darwin" {
		home, err := resolveUserHomeDir()
		if err == nil && strings.TrimSpace(home) != "" {
			return filepath.Join(home, "Library", "Logs", appDirName), nil
		}
	}

	cacheBase, err := resolveUserCacheDir()
	if err == nil && strings.TrimSpace(cacheBase) != "" {
		return filepath.Join(cacheBase, appDirName, logsDirName), nil
	}

	home, err := resolveUserHomeDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(home) == "" {
		return "", errors.New("unable to resolve home directory")
	}
	return filepath.Join(home, legacyDirName, logsDirName), nil
}

func LogsFile(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("log file name is required")
	}

	dir, err := LogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func appDataDir(primaryResolver func() (string, error)) (string, error) {
	if base, err := primaryResolver(); err == nil && strings.TrimSpace(base) != "" {
		return filepath.Join(base, appDirName), nil
	}

	home, err := resolveUserHomeDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(home) == "" {
		return "", errors.New("unable to resolve home directory")
	}
	return filepath.Join(home, legacyDirName), nil
}
