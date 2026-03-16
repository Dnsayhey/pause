//go:build linux

package linux

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"pause/internal/core/config"
)

func TestDesktopFileNameForAppID(t *testing.T) {
	cases := []struct {
		name  string
		appID string
		want  string
	}{
		{name: "empty", appID: "", want: "pause.desktop"},
		{name: "normal bundle id", appID: "com.pause.app", want: "com.pause.app.desktop"},
		{name: "trim and normalize", appID: "  COM.PAUSE/APP  ", want: "com.pause-app.desktop"},
		{name: "all invalid chars", appID: "???", want: "pause.desktop"},
	}

	for _, tc := range cases {
		got := desktopFileNameForAppID(tc.appID)
		if got != tc.want {
			t.Fatalf("%s: desktopFileNameForAppID(%q) = %q, want %q", tc.name, tc.appID, got, tc.want)
		}
	}
}

func TestDesktopEntryDisabled(t *testing.T) {
	enabled := `[Desktop Entry]
Type=Application
X-GNOME-Autostart-enabled=true
`
	if desktopEntryDisabled(enabled) {
		t.Fatalf("desktopEntryDisabled(enabled) = true, want false")
	}

	hidden := `[Desktop Entry]
Type=Application
Hidden=true
`
	if !desktopEntryDisabled(hidden) {
		t.Fatalf("desktopEntryDisabled(hidden) = false, want true")
	}

	disabled := `[Desktop Entry]
Type=Application
X-GNOME-Autostart-enabled=false
`
	if !desktopEntryDisabled(disabled) {
		t.Fatalf("desktopEntryDisabled(disabled) = false, want true")
	}
}

func TestBuildDesktopEntry(t *testing.T) {
	got := buildDesktopEntry("Pause", "/tmp/Pause App/pause", "com.pause.app")
	if !strings.Contains(got, "Name=Pause\n") {
		t.Fatalf("expected Name in desktop entry, got:\n%s", got)
	}
	if !strings.Contains(got, `Exec="/tmp/Pause App/pause"`+"\n") {
		t.Fatalf("expected quoted Exec in desktop entry, got:\n%s", got)
	}
	if !strings.Contains(got, "X-Pause-AppID=com.pause.app\n") {
		t.Fatalf("expected app id metadata in desktop entry, got:\n%s", got)
	}
}

func TestLinuxStartupManagerSetAndGetLaunchAtLogin(t *testing.T) {
	tmpDir := t.TempDir()
	oldResolveUserConfig := resolveUserConfig
	oldResolveExecutable := resolveExecutable
	resolveUserConfig = func() (string, error) { return tmpDir, nil }
	resolveExecutable = func() (string, error) { return "/opt/Pause App/pause", nil }
	t.Cleanup(func() {
		resolveUserConfig = oldResolveUserConfig
		resolveExecutable = oldResolveExecutable
	})

	mgr := linuxStartupManager{
		appID:           "com.pause.app",
		appName:         "Pause",
		desktopFileName: "pause-test.desktop",
	}

	if err := mgr.SetLaunchAtLogin(true); err != nil {
		t.Fatalf("SetLaunchAtLogin(true) error = %v", err)
	}

	enabled, err := mgr.GetLaunchAtLogin()
	if err != nil {
		t.Fatalf("GetLaunchAtLogin() error = %v", err)
	}
	if !enabled {
		t.Fatalf("GetLaunchAtLogin() = false, want true")
	}

	entryPath := filepath.Join(tmpDir, "autostart", "pause-test.desktop")
	raw, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read desktop entry error = %v", err)
	}
	if !strings.Contains(string(raw), `Exec="/opt/Pause App/pause"`) {
		t.Fatalf("unexpected Exec line in desktop entry:\n%s", string(raw))
	}

	if err := mgr.SetLaunchAtLogin(false); err != nil {
		t.Fatalf("SetLaunchAtLogin(false) error = %v", err)
	}

	enabled, err = mgr.GetLaunchAtLogin()
	if err != nil {
		t.Fatalf("GetLaunchAtLogin() after disable error = %v", err)
	}
	if enabled {
		t.Fatalf("GetLaunchAtLogin() after disable = true, want false")
	}
}

func TestLinuxNotifierFallbackToDBusSend(t *testing.T) {
	oldLookupExecutable := lookupExecutable
	oldRunCommandContext := runCommandContext
	lookupExecutable = func(name string) (string, error) {
		switch name {
		case "notify-send":
			return "", errors.New("missing")
		case "dbus-send":
			return "/usr/bin/dbus-send", nil
		default:
			return "", errors.New("unexpected command")
		}
	}
	runCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 0")
	}
	t.Cleanup(func() {
		lookupExecutable = oldLookupExecutable
		runCommandContext = oldRunCommandContext
	})

	n := linuxNotifier{appName: "Pause"}
	if err := n.ShowReminder("", ""); err != nil {
		t.Fatalf("ShowReminder() error = %v", err)
	}
}

func TestLinuxSoundPlayerUsesCanberraWhenAvailable(t *testing.T) {
	oldLookupExecutable := lookupExecutable
	oldStartCommand := startCommand
	lookupExecutable = func(name string) (string, error) {
		if name == "canberra-gtk-play" {
			return "/usr/bin/canberra-gtk-play", nil
		}
		return "", errors.New("missing")
	}
	var startedName string
	startCommand = func(name string, args ...string) *exec.Cmd {
		startedName = name
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() {
		lookupExecutable = oldLookupExecutable
		startCommand = oldStartCommand
	})

	player := linuxSoundPlayer{}
	if err := player.PlayBreakEnd(config.SoundSettings{Enabled: true, Volume: 70}); err != nil {
		t.Fatalf("PlayBreakEnd() error = %v", err)
	}
	if startedName != "canberra-gtk-play" {
		t.Fatalf("expected canberra-gtk-play, got %q", startedName)
	}
}

func TestPulseVolumeFromPercent(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{in: -1, want: 0},
		{in: 0, want: 0},
		{in: 50, want: 32768},
		{in: 100, want: 65536},
		{in: 120, want: 65536},
	}

	for _, tc := range cases {
		got := pulseVolumeFromPercent(tc.in)
		if got != tc.want {
			t.Fatalf("pulseVolumeFromPercent(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseIdleMillisecondsOutput(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{raw: "12345\n", want: 12},
		{raw: "(uint64 90123,)", want: 90},
		{raw: "0", want: 0},
	}

	for _, tc := range cases {
		got, err := parseIdleMillisecondsOutput([]byte(tc.raw))
		if err != nil {
			t.Fatalf("parseIdleMillisecondsOutput(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("parseIdleMillisecondsOutput(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestParseFirstUnsignedIntError(t *testing.T) {
	_, err := parseFirstUnsignedInt("no-digits-here")
	if err == nil {
		t.Fatalf("parseFirstUnsignedInt() expected error, got nil")
	}
}

func TestQueryLinuxIdleSecondsPrefersXPrintIdle(t *testing.T) {
	oldLookupExecutable := lookupExecutable
	oldRunCommandContext := runCommandContext
	oldReadEnv := readEnv
	lookupExecutable = func(name string) (string, error) {
		switch name {
		case "xprintidle":
			return "/usr/bin/xprintidle", nil
		default:
			return "", errors.New("missing")
		}
	}
	runCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "xprintidle" {
			return exec.CommandContext(ctx, "sh", "-c", "printf '42000\\n'")
		}
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	readEnv = func(key string) string {
		if key == "DISPLAY" {
			return ":0"
		}
		return ""
	}
	t.Cleanup(func() {
		lookupExecutable = oldLookupExecutable
		runCommandContext = oldRunCommandContext
		readEnv = oldReadEnv
	})

	sec, backend, err := queryLinuxIdleSeconds()
	if err != nil {
		t.Fatalf("queryLinuxIdleSeconds() error = %v", err)
	}
	if sec != 42 {
		t.Fatalf("queryLinuxIdleSeconds() sec = %d, want 42", sec)
	}
	if backend != "xprintidle" {
		t.Fatalf("queryLinuxIdleSeconds() backend = %q, want %q", backend, "xprintidle")
	}
}

func TestQueryLinuxIdleSecondsFallsBackToGnomeIdleMonitor(t *testing.T) {
	oldLookupExecutable := lookupExecutable
	oldRunCommandContext := runCommandContext
	oldReadEnv := readEnv
	lookupExecutable = func(name string) (string, error) {
		if name == "gdbus" {
			return "/usr/bin/gdbus", nil
		}
		return "", errors.New("missing")
	}
	runCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "gdbus" {
			return exec.CommandContext(ctx, "sh", "-c", "printf '(uint64 1500,)\\n'")
		}
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	readEnv = func(key string) string {
		if key == "DBUS_SESSION_BUS_ADDRESS" {
			return "unix:path=/tmp/fake"
		}
		return ""
	}
	t.Cleanup(func() {
		lookupExecutable = oldLookupExecutable
		runCommandContext = oldRunCommandContext
		readEnv = oldReadEnv
	})

	sec, backend, err := queryLinuxIdleSeconds()
	if err != nil {
		t.Fatalf("queryLinuxIdleSeconds() error = %v", err)
	}
	if sec != 1 {
		t.Fatalf("queryLinuxIdleSeconds() sec = %d, want 1", sec)
	}
	if backend != "gnome-idle-monitor" {
		t.Fatalf("queryLinuxIdleSeconds() backend = %q, want %q", backend, "gnome-idle-monitor")
	}
}

func TestDetectLinuxEnvironmentWayland(t *testing.T) {
	oldReadEnv := readEnv
	readEnv = func(key string) string {
		switch key {
		case "XDG_SESSION_TYPE":
			return "wayland"
		case "XDG_CURRENT_DESKTOP":
			return "GNOME"
		case "DESKTOP_SESSION":
			return "ubuntu"
		case "WAYLAND_DISPLAY":
			return "wayland-0"
		case "DISPLAY":
			return ""
		case "DBUS_SESSION_BUS_ADDRESS":
			return "unix:path=/tmp/bus"
		default:
			return ""
		}
	}
	t.Cleanup(func() {
		readEnv = oldReadEnv
	})

	env := detectLinuxEnvironment()
	if !env.hasWayland {
		t.Fatalf("detectLinuxEnvironment() hasWayland = false, want true")
	}
	if env.hasXDisplay {
		t.Fatalf("detectLinuxEnvironment() hasXDisplay = true, want false")
	}
	if env.sessionType != "wayland" {
		t.Fatalf("detectLinuxEnvironment() sessionType = %q, want %q", env.sessionType, "wayland")
	}
	if env.desktop != "gnome" {
		t.Fatalf("detectLinuxEnvironment() desktop = %q, want %q", env.desktop, "gnome")
	}
}

func TestDetectLinuxEnvironmentX11(t *testing.T) {
	oldReadEnv := readEnv
	readEnv = func(key string) string {
		switch key {
		case "XDG_SESSION_TYPE":
			return "x11"
		case "XDG_CURRENT_DESKTOP":
			return "KDE"
		case "DISPLAY":
			return ":0"
		default:
			return ""
		}
	}
	t.Cleanup(func() {
		readEnv = oldReadEnv
	})

	env := detectLinuxEnvironment()
	if env.hasWayland {
		t.Fatalf("detectLinuxEnvironment() hasWayland = true, want false")
	}
	if !env.hasXDisplay {
		t.Fatalf("detectLinuxEnvironment() hasXDisplay = false, want true")
	}
	if env.sessionType != "x11" {
		t.Fatalf("detectLinuxEnvironment() sessionType = %q, want %q", env.sessionType, "x11")
	}
}

func TestCommandAvailable(t *testing.T) {
	oldLookupExecutable := lookupExecutable
	lookupExecutable = func(name string) (string, error) {
		if name == "notify-send" {
			return "/usr/bin/notify-send", nil
		}
		return "", errors.New("missing")
	}
	t.Cleanup(func() {
		lookupExecutable = oldLookupExecutable
	})

	if !commandAvailable("notify-send") {
		t.Fatalf("commandAvailable(notify-send) = false, want true")
	}
	if commandAvailable("definitely-missing") {
		t.Fatalf("commandAvailable(definitely-missing) = true, want false")
	}
	if commandAvailable("  ") {
		t.Fatalf("commandAvailable(blank) = true, want false")
	}
}
