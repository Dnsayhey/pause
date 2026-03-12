//go:build darwin

package platform

import (
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
