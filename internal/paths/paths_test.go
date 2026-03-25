package paths

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPathResolvers_ConfigCacheLogs(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	configFile, err := ConfigFile("settings.json")
	if err != nil {
		t.Fatalf("ConfigFile() err=%v", err)
	}
	if want := filepath.Join("base", "config", "Pause", "settings.json"); configFile != want {
		t.Fatalf("ConfigFile()=%q want=%q", configFile, want)
	}

	cacheDir, err := CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() err=%v", err)
	}
	if want := filepath.Join("base", "cache", "Pause"); cacheDir != want {
		t.Fatalf("CacheDir()=%q want=%q", cacheDir, want)
	}

	logsDir, err := LogsDir()
	if err != nil {
		t.Fatalf("LogsDir() err=%v", err)
	}
	if want := filepath.Join("base", "cache", "Pause", "logs"); logsDir != want {
		t.Fatalf("LogsDir()=%q want=%q", logsDir, want)
	}
}

func TestPathResolvers_Fallbacks(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return "", errors.New("no config") },
		func() (string, error) { return "", errors.New("no cache") },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	configDir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() err=%v", err)
	}
	if want := filepath.Join("home", "user", ".pause"); configDir != want {
		t.Fatalf("ConfigDir()=%q want=%q", configDir, want)
	}

	logsDir, err := LogsDir()
	if err != nil {
		t.Fatalf("LogsDir() err=%v", err)
	}
	if want := filepath.Join("home", "user", ".pause", "logs"); logsDir != want {
		t.Fatalf("LogsDir()=%q want=%q", logsDir, want)
	}
}

func TestPathResolvers_RejectBlankNames(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	if _, err := ConfigFile(" "); err == nil {
		t.Fatalf("expected ConfigFile() reject blank name")
	}
	if _, err := LogsFile(" "); err == nil {
		t.Fatalf("expected LogsFile() reject blank name")
	}
}

func stubResolvers(
	config func() (string, error),
	cache func() (string, error),
	home func() (string, error),
	goos func() string,
) func() {
	oldConfig := resolveUserConfigDir
	oldCache := resolveUserCacheDir
	oldHome := resolveUserHomeDir
	oldGOOS := resolveGOOS

	resolveUserConfigDir = config
	resolveUserCacheDir = cache
	resolveUserHomeDir = home
	resolveGOOS = goos

	return func() {
		resolveUserConfigDir = oldConfig
		resolveUserCacheDir = oldCache
		resolveUserHomeDir = oldHome
		resolveGOOS = oldGOOS
	}
}
