package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pause/internal/paths"
)

const (
	logFileName = "app.log"
	maxLogSize  = 2 * 1024 * 1024
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Sink func(level Level, message string)

type logger struct {
	mu    sync.Mutex
	level Level
	path  string
	file  *os.File
	sink  Sink
}

var (
	once sync.Once
	base *logger
)

func Debugf(format string, args ...any) { logf(LevelDebug, format, args...) }
func Infof(format string, args ...any)  { logf(LevelInfo, format, args...) }
func Warnf(format string, args ...any)  { logf(LevelWarn, format, args...) }
func Errorf(format string, args ...any) { logf(LevelError, format, args...) }

func SetSink(sink Sink) {
	l := get()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sink = sink
}

func ClearSink() {
	SetSink(nil)
}

func logf(level Level, format string, args ...any) {
	l := get()
	if level < l.level {
		return
	}

	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format(time.RFC3339Nano), level.String(), msg)

	l.mu.Lock()
	if l.file != nil {
		l.rotateIfNeededLocked(int64(len(line) + 1))
		_, _ = l.file.WriteString(line + "\n")
	}
	sink := l.sink
	l.mu.Unlock()

	if sink != nil {
		sink(level, msg)
	}
}

func get() *logger {
	once.Do(func() {
		base = initLogger()
	})
	return base
}

func initLogger() *logger {
	l := &logger{
		level: parseLogLevel(),
	}

	logPath, err := paths.LogsFile(logFileName)
	if err != nil {
		return l
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return l
	}
	rotateIfNeeded(logPath)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return l
	}
	l.path = logPath
	l.file = f
	_, _ = f.WriteString(time.Now().Format(time.RFC3339Nano) + " [INFO] ---- logger started pid=" + fmt.Sprint(os.Getpid()) + " level=" + l.level.String() + " ----\n")
	return l
}

func parseLogLevel() Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PAUSE_LOG_LEVEL"))) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "info", "":
		return LevelInfo
	default:
		return LevelInfo
	}
}

func rotateIfNeeded(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxLogSize {
		return
	}

	backupPath := path + ".1"
	_ = os.Remove(backupPath)
	_ = os.Rename(path, backupPath)
}

func (l *logger) rotateIfNeededLocked(pendingWriteBytes int64) {
	if l.file == nil || l.path == "" {
		return
	}

	info, err := l.file.Stat()
	if err != nil {
		return
	}
	if info.Size()+pendingWriteBytes < maxLogSize {
		return
	}

	if err := l.file.Close(); err != nil {
		return
	}

	backupPath := l.path + ".1"
	_ = os.Remove(backupPath)
	_ = os.Rename(l.path, backupPath)

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		l.file = nil
		return
	}
	l.file = f
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}
