//go:build darwin

package platform

import (
	"errors"
	"strings"
	"testing"
)

func TestParseDarwinIdleSeconds(t *testing.T) {
	raw := []byte(`| |   "HIDIdleTime" = 301000000000`)
	sec, err := parseDarwinIdleSeconds(raw)
	if err != nil {
		t.Fatalf("parseDarwinIdleSeconds() error = %v", err)
	}
	if sec != 301 {
		t.Fatalf("expected 301 seconds, got %d", sec)
	}
}

func TestApplescriptQuote(t *testing.T) {
	got := applescriptQuote(`hello "pause" \\ world`)
	if !strings.HasPrefix(got, "\"") || !strings.HasSuffix(got, "\"") {
		t.Fatalf("expected quoted applescript string, got %q", got)
	}
	if !strings.Contains(got, `\"pause\"`) {
		t.Fatalf("expected embedded quotes escaped, got %q", got)
	}
	if !strings.Contains(got, `\\\\`) {
		t.Fatalf("expected backslashes escaped, got %q", got)
	}
}

func TestLaunchAgentPlistEscapesXML(t *testing.T) {
	content := launchAgentPlist("com.pause.app", "/tmp/pause<&>\"'.bin")
	checks := []string{
		"com.pause.app",
		"/tmp/pause&lt;&amp;&gt;&quot;&apos;.bin",
	}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Fatalf("launchAgentPlist missing %q", c)
		}
	}
}

func TestValidateStartupExecutablePath(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "normal app path",
			path:    "/Applications/Pause.app/Contents/MacOS/Pause",
			wantErr: false,
		},
		{
			name:    "mounted volume path",
			path:    "/Volumes/Pause/Pause.app/Contents/MacOS/Pause",
			wantErr: true,
		},
		{
			name:    "app translocation path",
			path:    "/private/var/folders/xx/AppTranslocation/ABC/Pause.app/Contents/MacOS/Pause",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		err := validateStartupExecutablePath(tc.path)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: expected no error, got %v", tc.name, err)
		}
	}
}

func TestIsLaunchctlAlreadyLoadedError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "already bootstrapped", err: errors.New("launchctl bootstrap failed: service already bootstrapped"), want: true},
		{name: "already loaded", err: errors.New("launchctl bootstrap failed: service already loaded"), want: true},
		{name: "other", err: errors.New("permission denied"), want: false},
	}
	for _, tc := range cases {
		got := isLaunchctlAlreadyLoadedError(tc.err)
		if got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
