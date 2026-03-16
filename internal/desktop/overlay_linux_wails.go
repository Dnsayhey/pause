//go:build wails && linux

package desktop

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"pause/internal/logx"
)

type linuxOverlayBackend string

const (
	linuxOverlayBackendNone     linuxOverlayBackend = ""
	linuxOverlayBackendYAD      linuxOverlayBackend = "yad"
	linuxOverlayBackendZenity   linuxOverlayBackend = "zenity"
	linuxOverlayBackendXMessage linuxOverlayBackend = "xmessage"
)

var (
	lookupOverlayExecutable = exec.LookPath
	startOverlayCommand     = exec.Command
	readOverlayEnv          = os.Getenv
)

type linuxBreakOverlayController struct {
	mu sync.Mutex

	onSkip func()

	native  bool
	backend linuxOverlayBackend

	cmd                 *exec.Cmd
	allowSkip           bool
	skipButtonTitle     string
	countdownText       string
	theme               string
	suppressedExitByPID map[int]struct{}
}

func NewBreakOverlayController() BreakOverlayController {
	c := &linuxBreakOverlayController{
		suppressedExitByPID: map[int]struct{}{},
	}
	c.native, c.backend = resolveLinuxOverlaySupport()
	return c
}

func (c *linuxBreakOverlayController) Init(onSkip func()) {
	c.mu.Lock()
	c.onSkip = onSkip
	c.mu.Unlock()
}

func (c *linuxBreakOverlayController) Show(allowSkip bool, skipButtonTitle string, countdownText string, theme string) bool {
	c.mu.Lock()
	if !c.native || c.backend == linuxOverlayBackendNone {
		c.mu.Unlock()
		return false
	}

	skipButtonTitle = overlayFallbackText(skipButtonTitle, "Emergency Skip")
	countdownText = overlayFallbackText(countdownText, "Time to rest")
	theme = normalizeOverlayTheme(theme)

	restart := c.cmd == nil ||
		c.allowSkip != allowSkip ||
		c.skipButtonTitle != skipButtonTitle ||
		c.theme != theme ||
		overlayCountdownBucket(c.countdownText) != overlayCountdownBucket(countdownText)

	c.allowSkip = allowSkip
	c.skipButtonTitle = skipButtonTitle
	c.countdownText = countdownText
	c.theme = theme
	backend := c.backend
	activeCmd := c.cmd
	c.mu.Unlock()

	if !restart && overlayCommandAlive(activeCmd) {
		return true
	}

	if activeCmd != nil {
		c.stopOverlayCommand(activeCmd)
	}

	cmd, err := launchLinuxOverlayCommand(backend, allowSkip, skipButtonTitle, countdownText, theme)
	if err != nil {
		logx.Warnf("linux.overlay_show_failed backend=%s err=%v", backend, err)
		return false
	}

	c.mu.Lock()
	c.cmd = cmd
	cb := c.onSkip
	c.mu.Unlock()

	go c.waitOverlayCommand(cmd, allowSkip, cb)
	return true
}

func (c *linuxBreakOverlayController) Hide() {
	c.mu.Lock()
	activeCmd := c.cmd
	c.cmd = nil
	c.mu.Unlock()

	if activeCmd != nil {
		c.stopOverlayCommand(activeCmd)
	}
}

func (c *linuxBreakOverlayController) Destroy() {
	c.Hide()
}

func (c *linuxBreakOverlayController) IsNative() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.native
}

func (c *linuxBreakOverlayController) waitOverlayCommand(cmd *exec.Cmd, allowSkip bool, onSkip func()) {
	err := cmd.Wait()
	pid := overlayCommandPID(cmd)
	exitCode := overlayExitCode(err)

	c.mu.Lock()
	if c.cmd == cmd {
		c.cmd = nil
	}
	suppressed := c.consumeSuppressedExitLocked(pid)
	c.mu.Unlock()

	if suppressed {
		return
	}

	logx.Debugf("linux.overlay_exit pid=%d code=%d err=%v", pid, exitCode, err)
	if allowSkip && exitCode == 0 && onSkip != nil {
		onSkip()
	}
}

func (c *linuxBreakOverlayController) stopOverlayCommand(cmd *exec.Cmd) {
	pid := overlayCommandPID(cmd)
	if pid > 0 {
		c.mu.Lock()
		c.suppressedExitByPID[pid] = struct{}{}
		c.mu.Unlock()
	}
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
}

func (c *linuxBreakOverlayController) consumeSuppressedExitLocked(pid int) bool {
	if pid <= 0 {
		return false
	}
	if _, ok := c.suppressedExitByPID[pid]; !ok {
		return false
	}
	delete(c.suppressedExitByPID, pid)
	return true
}

func resolveLinuxOverlaySupport() (bool, linuxOverlayBackend) {
	sessionType := normalizeOverlayLower(readOverlayEnv("XDG_SESSION_TYPE"), "unknown")
	display := strings.TrimSpace(readOverlayEnv("DISPLAY"))
	waylandDisplay := strings.TrimSpace(readOverlayEnv("WAYLAND_DISPLAY"))

	isWayland := sessionType == "wayland" || waylandDisplay != ""
	if isWayland {
		logx.Warnf("linux.overlay_native_disabled session_type=%s reason=wayland_limited fallback=frontend_overlay", sessionType)
		return false, linuxOverlayBackendNone
	}

	if display == "" {
		logx.Warnf("linux.overlay_native_disabled session_type=%s reason=display_missing fallback=frontend_overlay", sessionType)
		return false, linuxOverlayBackendNone
	}

	if commandExists("yad") {
		logx.Infof("linux.overlay_native_enabled backend=yad session_type=%s", sessionType)
		return true, linuxOverlayBackendYAD
	}
	if commandExists("zenity") {
		logx.Infof("linux.overlay_native_enabled backend=zenity session_type=%s", sessionType)
		return true, linuxOverlayBackendZenity
	}
	if commandExists("xmessage") {
		logx.Infof("linux.overlay_native_enabled backend=xmessage session_type=%s", sessionType)
		return true, linuxOverlayBackendXMessage
	}

	logx.Warnf("linux.overlay_native_disabled session_type=%s reason=no_overlay_backend fallback=frontend_overlay", sessionType)
	return false, linuxOverlayBackendNone
}

func launchLinuxOverlayCommand(
	backend linuxOverlayBackend,
	allowSkip bool,
	skipButtonTitle string,
	countdownText string,
	theme string,
) (*exec.Cmd, error) {
	args, err := buildLinuxOverlayArgs(backend, allowSkip, skipButtonTitle, countdownText, theme)
	if err != nil {
		return nil, err
	}

	cmd := startOverlayCommand(string(backend), args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func buildLinuxOverlayArgs(
	backend linuxOverlayBackend,
	allowSkip bool,
	skipButtonTitle string,
	countdownText string,
	theme string,
) ([]string, error) {
	text := overlayDisplayText(countdownText)

	switch backend {
	case linuxOverlayBackendYAD:
		args := []string{
			"--title=Pause",
			"--on-top",
			"--skip-taskbar",
			"--sticky",
			"--undecorated",
			"--fullscreen",
			"--center",
			"--text-align=center",
			"--text=" + text,
		}
		if allowSkip {
			args = append(args, "--button="+overlayEscapeButtonLabel(skipButtonTitle)+":0")
		} else {
			args = append(args, "--no-buttons")
		}
		if theme == "light" {
			args = append(args, "--borders=6")
		}
		return args, nil

	case linuxOverlayBackendZenity:
		if allowSkip {
			return []string{
				"--question",
				"--title=Pause",
				"--text=" + text,
				"--ok-label=" + overlayEscapeButtonLabel(skipButtonTitle),
				"--cancel-label=Close",
				"--width=460",
				"--height=180",
			}, nil
		}
		return []string{
			"--info",
			"--title=Pause",
			"--text=" + text,
			"--no-wrap",
			"--width=460",
			"--height=180",
		}, nil

	case linuxOverlayBackendXMessage:
		buttons := "Close:1"
		if allowSkip {
			buttons = overlayEscapeButtonLabel(skipButtonTitle) + ":0"
		}
		return []string{
			"-center",
			"-name",
			"Pause",
			"-buttons",
			buttons,
			text,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported linux overlay backend: %q", backend)
	}
}

func overlayDisplayText(countdownText string) string {
	return overlayFallbackText(strings.TrimSpace(countdownText), "Time to rest")
}

func overlayEscapeButtonLabel(title string) string {
	title = overlayFallbackText(title, "Emergency Skip")
	title = strings.ReplaceAll(title, ":", "")
	title = strings.TrimSpace(title)
	if title == "" {
		return "Emergency Skip"
	}
	return title
}

func overlayFallbackText(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func overlayCommandAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return cmd.ProcessState == nil
}

func overlayCommandPID(cmd *exec.Cmd) int {
	if cmd == nil || cmd.Process == nil {
		return 0
	}
	return cmd.Process.Pid
}

func overlayExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errorsAs(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func overlayCountdownBucket(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if len(text) >= 5 && text[2] == ':' {
		// Overlay backends are separate processes; refreshing once per minute avoids
		// visibly restarting the dialog every second.
		return text[:2]
	}
	if len(text) >= 5 {
		return text[:5]
	}
	return text
}

func normalizeOverlayTheme(theme string) string {
	theme = strings.ToLower(strings.TrimSpace(theme))
	if theme == "light" {
		return "light"
	}
	return "dark"
}

func normalizeOverlayLower(v, fallback string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return fallback
	}
	return v
}

func commandExists(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	_, err := lookupOverlayExecutable(name)
	return err == nil
}

// errorsAs is wrapped for easier stubbing in tests.
var errorsAs = func(err error, target any) bool {
	return errors.As(err, target)
}
