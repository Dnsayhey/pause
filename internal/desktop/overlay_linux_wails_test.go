//go:build wails && linux

package desktop

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveLinuxOverlaySupportWaylandDisabled(t *testing.T) {
	oldReadOverlayEnv := readOverlayEnv
	oldLookupOverlayExecutable := lookupOverlayExecutable
	readOverlayEnv = func(key string) string {
		switch key {
		case "XDG_SESSION_TYPE":
			return "wayland"
		case "WAYLAND_DISPLAY":
			return "wayland-0"
		case "DISPLAY":
			return ":0"
		default:
			return ""
		}
	}
	lookupOverlayExecutable = func(name string) (string, error) {
		return "", errors.New("not used")
	}
	t.Cleanup(func() {
		readOverlayEnv = oldReadOverlayEnv
		lookupOverlayExecutable = oldLookupOverlayExecutable
	})

	native, backend := resolveLinuxOverlaySupport()
	if native {
		t.Fatalf("resolveLinuxOverlaySupport() native = true, want false")
	}
	if backend != linuxOverlayBackendNone {
		t.Fatalf("resolveLinuxOverlaySupport() backend = %q, want empty", backend)
	}
}

func TestResolveLinuxOverlaySupportX11YAD(t *testing.T) {
	oldReadOverlayEnv := readOverlayEnv
	oldLookupOverlayExecutable := lookupOverlayExecutable
	readOverlayEnv = func(key string) string {
		switch key {
		case "XDG_SESSION_TYPE":
			return "x11"
		case "DISPLAY":
			return ":0"
		default:
			return ""
		}
	}
	lookupOverlayExecutable = func(name string) (string, error) {
		if name == "yad" {
			return "/usr/bin/yad", nil
		}
		return "", errors.New("missing")
	}
	t.Cleanup(func() {
		readOverlayEnv = oldReadOverlayEnv
		lookupOverlayExecutable = oldLookupOverlayExecutable
	})

	native, backend := resolveLinuxOverlaySupport()
	if !native {
		t.Fatalf("resolveLinuxOverlaySupport() native = false, want true")
	}
	if backend != linuxOverlayBackendYAD {
		t.Fatalf("resolveLinuxOverlaySupport() backend = %q, want %q", backend, linuxOverlayBackendYAD)
	}
}

func TestResolveLinuxOverlaySupportX11NoBackend(t *testing.T) {
	oldReadOverlayEnv := readOverlayEnv
	oldLookupOverlayExecutable := lookupOverlayExecutable
	readOverlayEnv = func(key string) string {
		if key == "DISPLAY" {
			return ":0"
		}
		return "x11"
	}
	lookupOverlayExecutable = func(name string) (string, error) {
		return "", errors.New("missing")
	}
	t.Cleanup(func() {
		readOverlayEnv = oldReadOverlayEnv
		lookupOverlayExecutable = oldLookupOverlayExecutable
	})

	native, backend := resolveLinuxOverlaySupport()
	if native {
		t.Fatalf("resolveLinuxOverlaySupport() native = true, want false")
	}
	if backend != linuxOverlayBackendNone {
		t.Fatalf("resolveLinuxOverlaySupport() backend = %q, want empty", backend)
	}
}

func TestBuildLinuxOverlayArgsYAD(t *testing.T) {
	args, err := buildLinuxOverlayArgs(
		linuxOverlayBackendYAD,
		true,
		"Emergency Skip",
		"01:23",
		"dark",
	)
	if err != nil {
		t.Fatalf("buildLinuxOverlayArgs() error = %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--fullscreen") {
		t.Fatalf("expected fullscreen args, got: %s", joined)
	}
	if !strings.Contains(joined, "--button=Emergency Skip:0") {
		t.Fatalf("expected skip button args, got: %s", joined)
	}
}

func TestBuildLinuxOverlayArgsNoSkip(t *testing.T) {
	args, err := buildLinuxOverlayArgs(
		linuxOverlayBackendYAD,
		false,
		"Emergency Skip",
		"01:23",
		"dark",
	)
	if err != nil {
		t.Fatalf("buildLinuxOverlayArgs() error = %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--no-buttons") {
		t.Fatalf("expected no-buttons args, got: %s", joined)
	}
}
