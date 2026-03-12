package diag

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	traceFileName = "trace.log"
	maxTraceSize  = 2 * 1024 * 1024
)

type traceWriter struct {
	mu      sync.Mutex
	enabled bool
	file    *os.File
}

var (
	once   sync.Once
	writer *traceWriter
)

// Logf appends a single diagnostics line to ~/.pause/trace.log.
// Logging is disabled by default and enabled with PAUSE_DEBUG_LOG=1.
func Logf(format string, args ...any) {
	t := getWriter()
	if t == nil || !t.enabled {
		return
	}

	line := fmt.Sprintf(format, args...)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.file == nil {
		return
	}
	_, _ = t.file.WriteString(time.Now().Format(time.RFC3339Nano) + " " + line + "\n")
}

func getWriter() *traceWriter {
	once.Do(func() {
		writer = initWriter()
	})
	return writer
}

func initWriter() *traceWriter {
	if !isEnabledByEnv() {
		return &traceWriter{enabled: false}
	}

	path, err := tracePath()
	if err != nil {
		return &traceWriter{enabled: false}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &traceWriter{enabled: false}
	}

	rotateIfNeeded(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return &traceWriter{enabled: false}
	}

	t := &traceWriter{
		enabled: true,
		file:    f,
	}
	_, _ = f.WriteString(time.Now().Format(time.RFC3339Nano) + " ---- trace started pid=" + strconv.Itoa(os.Getpid()) + " ----\n")
	return t
}

func isEnabledByEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("PAUSE_DEBUG_LOG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func tracePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pause", traceFileName), nil
}

func rotateIfNeeded(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxTraceSize {
		return
	}

	backupPath := path + ".1"
	_ = os.Remove(backupPath)
	_ = os.Rename(path, backupPath)
}
