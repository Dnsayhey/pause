package paths

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestConfigFileUsesSystemConfigDir(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	got, err := ConfigFile("settings.json")
	if err != nil {
		t.Fatalf("ConfigFile() error = %v", err)
	}
	want := filepath.Join("base", "config", "Pause", "settings.json")
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestConfigDirFallsBackToLegacyHomeDir(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return "", errors.New("unavailable") },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	want := filepath.Join("home", "user", ".pause")
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestCacheDirUsesSystemCacheDir(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	got, err := CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() error = %v", err)
	}
	want := filepath.Join("base", "cache", "Pause")
	if got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}

func TestConfigFileRequiresName(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	if _, err := ConfigFile(" "); err == nil {
		t.Fatalf("expected ConfigFile() to reject empty name")
	}
}

func TestLogsDirOnDarwinUsesLibraryLogs(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "darwin" },
	)
	defer restore()

	got, err := LogsDir()
	if err != nil {
		t.Fatalf("LogsDir() error = %v", err)
	}
	want := filepath.Join("home", "user", "Library", "Logs", "Pause")
	if got != want {
		t.Fatalf("LogsDir() = %q, want %q", got, want)
	}
}

func TestLogsDirOnLinuxUsesCacheLogs(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	got, err := LogsDir()
	if err != nil {
		t.Fatalf("LogsDir() error = %v", err)
	}
	want := filepath.Join("base", "cache", "Pause", "logs")
	if got != want {
		t.Fatalf("LogsDir() = %q, want %q", got, want)
	}
}

func TestLogsDirFallsBackToLegacyOnCacheFailure(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return "", errors.New("cache unavailable") },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	got, err := LogsDir()
	if err != nil {
		t.Fatalf("LogsDir() error = %v", err)
	}
	want := filepath.Join("home", "user", ".pause", "logs")
	if got != want {
		t.Fatalf("LogsDir() = %q, want %q", got, want)
	}
}

func TestLogsFileRequiresName(t *testing.T) {
	restore := stubResolvers(
		func() (string, error) { return filepath.Join("base", "config"), nil },
		func() (string, error) { return filepath.Join("base", "cache"), nil },
		func() (string, error) { return filepath.Join("home", "user"), nil },
		func() string { return "linux" },
	)
	defer restore()

	if _, err := LogsFile(" "); err == nil {
		t.Fatalf("expected LogsFile() to reject empty name")
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
