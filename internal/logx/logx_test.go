package logx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLogLevelFromEnv(t *testing.T) {
	t.Setenv("PAUSE_LOG_LEVEL", "warn")

	if got := parseLogLevel(); got != LevelWarn {
		t.Fatalf("parseLogLevel() = %v, want %v", got, LevelWarn)
	}
}

func TestParseLogLevelDefaultsToInfo(t *testing.T) {
	t.Setenv("PAUSE_LOG_LEVEL", "")

	if got := parseLogLevel(); got != LevelInfo {
		t.Fatalf("parseLogLevel() = %v, want %v", got, LevelInfo)
	}
}

func TestRotateIfNeededLockedKeepsSingleBackup(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	payload := make([]byte, maxLogSize)
	if err := os.WriteFile(logPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile(log) error = %v", err)
	}

	backupPath := logPath + ".1"
	if err := os.WriteFile(backupPath, []byte("old backup"), 0o644); err != nil {
		t.Fatalf("WriteFile(backup) error = %v", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	l := &logger{path: logPath, file: f}
	l.rotateIfNeededLocked(1)

	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Stat(backup) error = %v", err)
	}
	if backupInfo.Size() != int64(maxLogSize) {
		t.Fatalf("backup size = %d, want %d", backupInfo.Size(), maxLogSize)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Stat(log) error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("new log size = %d, want 0", info.Size())
	}
}
