//go:build linux

package linux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"pause/internal/core/config"
	"pause/internal/logx"
	"pause/internal/meta"
	"pause/internal/platform/api"
)

const (
	linuxAppName                = "Pause"
	linuxReminderTimeoutMS      = "5000"
	linuxDefaultDesktopFileName = "pause.desktop"
	linuxDefaultSoundFile       = "/usr/share/sounds/freedesktop/stereo/complete.oga"
	linuxFallbackSoundFile      = "/usr/share/sounds/alsa/Front_Center.wav"
	linuxIdleSampleTTL          = 2 * time.Second
	linuxIdleProbeTimeout       = 300 * time.Millisecond
)

var (
	runCommandContext   = exec.CommandContext
	startCommand        = exec.Command
	lookupExecutable    = exec.LookPath
	resolveExecutable   = os.Executable
	resolveUserConfig   = os.UserConfigDir
	readEnv             = os.Getenv
	linuxCommandTimeout = 5 * time.Second
	logLinuxEnvOnce     sync.Once
)

type linuxIdleProvider struct {
	mu                sync.Mutex
	lastSampleAt      time.Time
	lastIdleSec       int
	lastBackend       string
	loggedUnavailable bool
}

type linuxNotifier struct {
	appName string
}

type linuxSoundPlayer struct{}

type linuxStartupManager struct {
	appID           string
	appName         string
	desktopFileName string
}

type linuxEnvironment struct {
	sessionType    string
	desktop        string
	desktopSession string
	hasWayland     bool
	hasXDisplay    bool
	hasDBusSession bool
}

func NewAdapters(appID string) api.Adapters {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = meta.EffectiveAppBundleID()
	}
	logLinuxEnvironmentOnce()

	return api.Adapters{
		IdleProvider: &linuxIdleProvider{},
		Notifier:     linuxNotifier{appName: linuxAppName},
		SoundPlayer:  linuxSoundPlayer{},
		StartupManager: linuxStartupManager{
			appID:           appID,
			appName:         linuxAppName,
			desktopFileName: desktopFileNameForAppID(appID),
		},
	}
}

func logLinuxEnvironmentOnce() {
	logLinuxEnvOnce.Do(func() {
		env := detectLinuxEnvironment()
		logx.Infof(
			"linux.env session_type=%s desktop=%s desktop_session=%s has_wayland=%t has_xdisplay=%t has_dbus_session=%t",
			env.sessionType,
			env.desktop,
			env.desktopSession,
			env.hasWayland,
			env.hasXDisplay,
			env.hasDBusSession,
		)
		if env.hasWayland {
			logx.Warnf("linux.wayland_limited overlay_enforcement=best_effort reason=wayland_security_model")
		}
		logx.Infof(
			"linux.capabilities idle[xprintidle=%t,xssstate=%t,gnome_idle_monitor=%t] notify[notify_send=%t,dbus_send=%t] sound[canberra=%t,paplay=%t,aplay=%t] startup[desktop_entry=true]",
			commandAvailable("xprintidle"),
			commandAvailable("xssstate"),
			commandAvailable("gdbus"),
			commandAvailable("notify-send"),
			commandAvailable("dbus-send"),
			commandAvailable("canberra-gtk-play"),
			commandAvailable("paplay"),
			commandAvailable("aplay"),
		)
	})
}

func detectLinuxEnvironment() linuxEnvironment {
	sessionType := normalizeLower(readEnv("XDG_SESSION_TYPE"), "unknown")
	desktop := normalizeLower(readEnv("XDG_CURRENT_DESKTOP"), "unknown")
	desktopSession := normalizeLower(readEnv("DESKTOP_SESSION"), "unknown")
	waylandDisplay := strings.TrimSpace(readEnv("WAYLAND_DISPLAY"))
	xDisplay := strings.TrimSpace(readEnv("DISPLAY"))
	dbusSession := strings.TrimSpace(readEnv("DBUS_SESSION_BUS_ADDRESS"))

	hasWayland := sessionType == "wayland" || waylandDisplay != ""
	hasXDisplay := sessionType == "x11" || xDisplay != ""
	return linuxEnvironment{
		sessionType:    sessionType,
		desktop:        desktop,
		desktopSession: desktopSession,
		hasWayland:     hasWayland,
		hasXDisplay:    hasXDisplay,
		hasDBusSession: dbusSession != "",
	}
}

func commandAvailable(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	_, err := lookupExecutable(name)
	return err == nil
}

func normalizeLower(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func (p *linuxIdleProvider) CurrentIdleSeconds() int {
	now := time.Now()

	p.mu.Lock()
	if !p.lastSampleAt.IsZero() {
		age := now.Sub(p.lastSampleAt)
		if age <= linuxIdleSampleTTL {
			idleSec := p.lastIdleSec + int(age.Seconds())
			p.mu.Unlock()
			return idleSec
		}
	}
	p.mu.Unlock()

	idleSec, backend, err := queryLinuxIdleSeconds()
	if err != nil {
		p.mu.Lock()
		if !p.loggedUnavailable {
			logx.Warnf("linux.idle_probe_unavailable err=%v", err)
			p.loggedUnavailable = true
		}
		if p.lastSampleAt.IsZero() {
			p.mu.Unlock()
			return 0
		}
		age := now.Sub(p.lastSampleAt)
		fallback := p.lastIdleSec + int(age.Seconds())
		p.mu.Unlock()
		return fallback
	}

	p.mu.Lock()
	p.lastSampleAt = now
	p.lastIdleSec = idleSec
	if backend != "" && backend != p.lastBackend {
		logx.Infof("linux.idle_probe_backend backend=%s", backend)
		p.lastBackend = backend
	}
	p.loggedUnavailable = false
	p.mu.Unlock()
	return idleSec
}

func queryLinuxIdleSeconds() (int, string, error) {
	var errs []string

	if sec, err := queryXPrintIdleSeconds(); err == nil {
		return sec, "xprintidle", nil
	} else {
		errs = append(errs, fmt.Sprintf("xprintidle: %v", err))
	}

	if sec, err := queryXSSStateIdleSeconds(); err == nil {
		return sec, "xssstate", nil
	} else {
		errs = append(errs, fmt.Sprintf("xssstate: %v", err))
	}

	if sec, err := queryGnomeIdleMonitorSeconds(); err == nil {
		return sec, "gnome-idle-monitor", nil
	} else {
		errs = append(errs, fmt.Sprintf("gnome-idle-monitor: %v", err))
	}

	if len(errs) == 0 {
		return 0, "", errors.New("no idle backend available")
	}
	return 0, "", errors.New(strings.Join(errs, "; "))
}

func queryXPrintIdleSeconds() (int, error) {
	if strings.TrimSpace(readEnv("DISPLAY")) == "" {
		return 0, errors.New("DISPLAY is empty")
	}
	if _, err := lookupExecutable("xprintidle"); err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), linuxIdleProbeTimeout)
	defer cancel()
	out, err := runCommandContext(ctx, "xprintidle").Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, errors.New("xprintidle timed out")
		}
		return 0, err
	}
	return parseIdleMillisecondsOutput(out)
}

func queryXSSStateIdleSeconds() (int, error) {
	if strings.TrimSpace(readEnv("DISPLAY")) == "" {
		return 0, errors.New("DISPLAY is empty")
	}
	if _, err := lookupExecutable("xssstate"); err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), linuxIdleProbeTimeout)
	defer cancel()
	out, err := runCommandContext(ctx, "xssstate", "-i").Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, errors.New("xssstate timed out")
		}
		return 0, err
	}
	return parseIdleMillisecondsOutput(out)
}

func queryGnomeIdleMonitorSeconds() (int, error) {
	if strings.TrimSpace(readEnv("DBUS_SESSION_BUS_ADDRESS")) == "" {
		return 0, errors.New("DBUS_SESSION_BUS_ADDRESS is empty")
	}
	if _, err := lookupExecutable("gdbus"); err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), linuxIdleProbeTimeout)
	defer cancel()
	out, err := runCommandContext(
		ctx,
		"gdbus",
		"call",
		"--session",
		"--dest", "org.gnome.Mutter.IdleMonitor",
		"--object-path", "/org/gnome/Mutter/IdleMonitor/Core",
		"--method", "org.gnome.Mutter.IdleMonitor.GetIdletime",
	).Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, errors.New("gdbus timed out")
		}
		return 0, err
	}
	return parseIdleMillisecondsOutput(out)
}

func (n linuxNotifier) ShowReminder(title, body string) error {
	title = fallbackText(title, n.appName)
	body = fallbackText(body, "Break started")

	var errs []string
	if err := notifyWithNotifySend(n.appName, title, body); err == nil {
		logx.Debugf("linux.notify backend=notify-send")
		return nil
	} else {
		logx.Debugf("linux.notify backend=notify-send err=%v", err)
		errs = append(errs, fmt.Sprintf("notify-send: %v", err))
	}

	if err := notifyWithDBusSend(n.appName, title, body); err == nil {
		logx.Debugf("linux.notify backend=dbus-send")
		return nil
	} else {
		logx.Debugf("linux.notify backend=dbus-send err=%v", err)
		errs = append(errs, fmt.Sprintf("dbus-send: %v", err))
	}

	wrapped := fmt.Errorf("linux reminder notification failed (%s)", strings.Join(errs, "; "))
	logx.Warnf("linux.notify_failed err=%v", wrapped)
	return wrapped
}

func notifyWithNotifySend(appName, title, body string) error {
	if _, err := lookupExecutable("notify-send"); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), linuxCommandTimeout)
	defer cancel()

	cmd := runCommandContext(
		ctx,
		"notify-send",
		"-a", fallbackText(appName, linuxAppName),
		"-u", "normal",
		"-t", linuxReminderTimeoutMS,
		title,
		body,
	)
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("notify-send timed out")
		}
		return err
	}
	return nil
}

func notifyWithDBusSend(appName, title, body string) error {
	if _, err := lookupExecutable("dbus-send"); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), linuxCommandTimeout)
	defer cancel()

	// org.freedesktop.Notifications.Notify(
	//   app_name, replaces_id, app_icon, summary, body, actions, hints, expire_timeout
	// )
	args := []string{
		"--session",
		"--dest=org.freedesktop.Notifications",
		"--type=method_call",
		"--print-reply",
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications.Notify",
		"string:" + fallbackText(appName, linuxAppName),
		"uint32:0",
		"string:",
		"string:" + title,
		"string:" + body,
		"array:string:",
		"dict:string:string:",
		"int32:" + linuxReminderTimeoutMS,
	}

	cmd := runCommandContext(ctx, "dbus-send", args...)
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("dbus-send timed out")
		}
		return err
	}
	return nil
}

func (linuxSoundPlayer) PlayBreakEnd(sound config.SoundSettings) error {
	if !sound.Enabled {
		return nil
	}

	var errs []string
	if err := playWithCanberra(); err == nil {
		logx.Debugf("linux.sound backend=canberra-gtk-play")
		return nil
	} else {
		logx.Debugf("linux.sound backend=canberra-gtk-play err=%v", err)
		errs = append(errs, fmt.Sprintf("canberra-gtk-play: %v", err))
	}

	if err := playWithPaplay(sound.Volume); err == nil {
		logx.Debugf("linux.sound backend=paplay")
		return nil
	} else {
		logx.Debugf("linux.sound backend=paplay err=%v", err)
		errs = append(errs, fmt.Sprintf("paplay: %v", err))
	}

	if err := playWithAplay(); err == nil {
		logx.Debugf("linux.sound backend=aplay")
		return nil
	} else {
		logx.Debugf("linux.sound backend=aplay err=%v", err)
		errs = append(errs, fmt.Sprintf("aplay: %v", err))
	}

	wrapped := fmt.Errorf("linux break-end sound failed (%s)", strings.Join(errs, "; "))
	logx.Warnf("linux.sound_failed err=%v", wrapped)
	return wrapped
}

func playWithCanberra() error {
	if _, err := lookupExecutable("canberra-gtk-play"); err != nil {
		return err
	}
	return startDetached("canberra-gtk-play", "-i", "complete", "-d", linuxAppName)
}

func playWithPaplay(volume int) error {
	if _, err := lookupExecutable("paplay"); err != nil {
		return err
	}
	soundFile, err := resolveSoundFile(linuxDefaultSoundFile, linuxFallbackSoundFile)
	if err != nil {
		return err
	}
	pulseVolume := pulseVolumeFromPercent(volume)
	return startDetached("paplay", fmt.Sprintf("--volume=%d", pulseVolume), soundFile)
}

func playWithAplay() error {
	if _, err := lookupExecutable("aplay"); err != nil {
		return err
	}
	soundFile, err := resolveSoundFile(linuxFallbackSoundFile, linuxDefaultSoundFile)
	if err != nil {
		return err
	}
	return startDetached("aplay", "-q", soundFile)
}

func resolveSoundFile(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("no compatible sound file found")
}

func pulseVolumeFromPercent(volume int) int {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	// PulseAudio uses 65536 as ~100%.
	return (volume * 65536) / 100
}

func startDetached(name string, args ...string) error {
	cmd := startCommand(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func (s linuxStartupManager) SetLaunchAtLogin(enabled bool) error {
	desktopFilePath, err := s.desktopEntryPath()
	if err != nil {
		return err
	}

	if !enabled {
		err = os.Remove(desktopFilePath)
		if errors.Is(err, os.ErrNotExist) {
			logx.Debugf("linux.startup disabled=true path=%s existed=false", desktopFilePath)
			return nil
		}
		if err == nil {
			logx.Debugf("linux.startup disabled=true path=%s existed=true", desktopFilePath)
		}
		return err
	}

	execPath, err := resolveExecutable()
	if err != nil {
		return err
	}
	execPath = strings.TrimSpace(execPath)
	if execPath == "" {
		return errors.New("empty executable path")
	}
	if !filepath.IsAbs(execPath) {
		execPath, err = filepath.Abs(execPath)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(desktopFilePath), 0o755); err != nil {
		return err
	}

	entry := buildDesktopEntry(s.appName, execPath, s.appID)
	if err := os.WriteFile(desktopFilePath, []byte(entry), 0o644); err != nil {
		return err
	}
	logx.Debugf("linux.startup enabled=true path=%s", desktopFilePath)
	return nil
}

func (s linuxStartupManager) GetLaunchAtLogin() (bool, error) {
	desktopFilePath, err := s.desktopEntryPath()
	if err != nil {
		return false, err
	}

	raw, err := os.ReadFile(desktopFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	enabled := !desktopEntryDisabled(string(raw))
	logx.Debugf("linux.startup query_enabled=%t path=%s", enabled, desktopFilePath)
	return enabled, nil
}

func (s linuxStartupManager) desktopEntryPath() (string, error) {
	configDir, err := resolveUserConfig()
	if err != nil {
		return "", err
	}
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return "", errors.New("user config directory is empty")
	}

	fileName := strings.TrimSpace(s.desktopFileName)
	if fileName == "" {
		fileName = linuxDefaultDesktopFileName
	}
	return filepath.Join(configDir, "autostart", fileName), nil
}

func buildDesktopEntry(appName, execPath, appID string) string {
	appName = fallbackText(appName, linuxAppName)
	execPath = strings.TrimSpace(execPath)
	appID = strings.TrimSpace(appID)

	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Version=1.0\n")
	b.WriteString("Name=" + appName + "\n")
	b.WriteString("Comment=Pause break reminder\n")
	b.WriteString("Exec=" + quoteDesktopExec(execPath) + "\n")
	b.WriteString("Terminal=false\n")
	b.WriteString("StartupNotify=false\n")
	b.WriteString("X-GNOME-Autostart-enabled=true\n")
	if appID != "" {
		b.WriteString("X-Pause-AppID=" + appID + "\n")
	}
	return b.String()
}

func quoteDesktopExec(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
	)
	return `"` + replacer.Replace(path) + `"`
}

func desktopEntryDisabled(raw string) bool {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.ToLower(strings.TrimSpace(parts[1]))
		if key == "hidden" && val == "true" {
			return true
		}
		if key == "x-gnome-autostart-enabled" && val == "false" {
			return true
		}
	}
	return false
}

func desktopFileNameForAppID(appID string) string {
	cleaned := strings.TrimSpace(strings.ToLower(appID))
	if cleaned == "" {
		return linuxDefaultDesktopFileName
	}

	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	name := strings.Trim(b.String(), ".-_")
	if name == "" {
		return linuxDefaultDesktopFileName
	}
	return name + ".desktop"
}

func parseIdleMillisecondsOutput(raw []byte) (int, error) {
	ms, err := parseFirstUnsignedInt(string(raw))
	if err != nil {
		return 0, err
	}
	return int(ms / 1000), nil
}

func parseFirstUnsignedInt(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("empty output")
	}

	start := -1
	end := -1
	for i, r := range raw {
		if r >= '0' && r <= '9' {
			if start == -1 {
				start = i
			}
			end = i + 1
			continue
		}
		if start != -1 {
			break
		}
	}
	if start == -1 || end == -1 {
		return 0, fmt.Errorf("no numeric value found in %q", raw)
	}

	n, err := strconv.ParseUint(raw[start:end], 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func fallbackText(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}
