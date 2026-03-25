package logx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLogLevel_FromEnv(t *testing.T) {
	t.Setenv("PAUSE_LOG_LEVEL", "warn")
	if got := parseLogLevel(); got != LevelWarn {
		t.Fatalf("parseLogLevel()=%v want=%v", got, LevelWarn)
	}
}

func TestParseLogLevel_DefaultInfo(t *testing.T) {
	t.Setenv("PAUSE_LOG_LEVEL", "")
	if got := parseLogLevel(); got != LevelInfo {
		t.Fatalf("parseLogLevel()=%v want=%v", got, LevelInfo)
	}
}

func TestRotateIfNeededLocked_ReplacesSingleBackup(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	payload := make([]byte, maxLogSize)
	if err := os.WriteFile(logPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile(log) err=%v", err)
	}
	backupPath := logPath + ".1"
	if err := os.WriteFile(backupPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile(backup) err=%v", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() err=%v", err)
	}
	defer f.Close()

	l := &logger{path: logPath, file: f}
	l.rotateIfNeededLocked(1)

	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Stat(backup) err=%v", err)
	}
	if backupInfo.Size() != int64(maxLogSize) {
		t.Fatalf("backup size mismatch got=%d want=%d", backupInfo.Size(), maxLogSize)
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Stat(log) err=%v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("log size mismatch got=%d want=0", info.Size())
	}
}
