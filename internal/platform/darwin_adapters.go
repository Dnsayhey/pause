//go:build darwin

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"pause/internal/config"
)

const (
	defaultMacSound = "/System/Library/Sounds/Glass.aiff"
	idleSampleTTL   = 2 * time.Second
	idleProbeTimeo  = 300 * time.Millisecond
)

var hidIdlePattern = regexp.MustCompile(`"HIDIdleTime" = ([0-9]+)`)

type darwinIdleProvider struct {
	mu           sync.Mutex
	lastSampleAt time.Time
	lastIdleSec  int
}

type darwinNotifier struct{}

type darwinSoundPlayer struct{}

type darwinStartupManager struct {
	appID string
}

func newDarwinAdapters(appID string) Adapters {
	if strings.TrimSpace(appID) == "" {
		appID = "com.pause.app"
	}
	return Adapters{
		IdleProvider:   &darwinIdleProvider{},
		Notifier:       darwinNotifier{},
		SoundPlayer:    darwinSoundPlayer{},
		StartupManager: darwinStartupManager{appID: appID},
	}
}

func (p *darwinIdleProvider) CurrentIdleSeconds() int {
	now := time.Now()

	p.mu.Lock()
	if !p.lastSampleAt.IsZero() {
		age := now.Sub(p.lastSampleAt)
		if age <= idleSampleTTL {
			idleSec := p.lastIdleSec + int(age.Seconds())
			p.mu.Unlock()
			return idleSec
		}
	}
	p.mu.Unlock()

	idleSec, ok := queryDarwinIdleSeconds()
	if !ok {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.lastSampleAt.IsZero() {
			return 0
		}
		age := now.Sub(p.lastSampleAt)
		return p.lastIdleSec + int(age.Seconds())
	}

	p.mu.Lock()
	p.lastSampleAt = now
	p.lastIdleSec = idleSec
	p.mu.Unlock()
	return idleSec
}

func queryDarwinIdleSeconds() (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), idleProbeTimeo)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ioreg", "-c", "IOHIDSystem").Output()
	if err != nil {
		return 0, false
	}
	idleSec, err := parseDarwinIdleSeconds(out)
	if err != nil {
		return 0, false
	}
	return idleSec, true
}

func parseDarwinIdleSeconds(raw []byte) (int, error) {
	match := hidIdlePattern.FindSubmatch(raw)
	if len(match) < 2 {
		return 0, errors.New("HIDIdleTime not found")
	}
	var ns int64
	_, err := fmt.Sscanf(string(match[1]), "%d", &ns)
	if err != nil {
		return 0, err
	}
	if ns <= 0 {
		return 0, nil
	}
	return int(ns / 1_000_000_000), nil
}

func (darwinNotifier) ShowReminder(title, body string) error {
	script := fmt.Sprintf("display notification %s with title %s", applescriptQuote(body), applescriptQuote(title))
	return exec.Command("osascript", "-e", script).Run()
}

func applescriptQuote(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return "\"" + escaped + "\""
}

func (darwinSoundPlayer) PlayBreakEnd(sound config.SoundSettings) error {
	if !sound.Enabled {
		return nil
	}
	volume := float64(sound.Volume) / 100.0
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	return exec.Command("afplay", "-v", fmt.Sprintf("%.2f", volume), defaultMacSound).Start()
}

func (s darwinStartupManager) SetLaunchAtLogin(enabled bool) error {
	plistPath, err := launchAgentPath(s.appID)
	if err != nil {
		return err
	}

	if !enabled {
		_ = exec.Command("launchctl", "unload", "-w", plistPath).Run()
		if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}

	content := launchAgentPlist(s.appID, execPath)
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return err
	}

	_ = exec.Command("launchctl", "unload", "-w", plistPath).Run()
	_ = exec.Command("launchctl", "load", "-w", plistPath).Run()
	return nil
}

func launchAgentPath(appID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", appID+".plist"), nil
}

func launchAgentPlist(appID, execPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
`, xmlEscape(appID), xmlEscape(execPath))
}

func xmlEscape(v string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(v)
}
