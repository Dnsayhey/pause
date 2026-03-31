package logx

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetGlobalLoggerState() {
	if base != nil {
		base.mu.Lock()
		file := base.file
		base.file = nil
		base.sink = nil
		base.mu.Unlock()
		if file != nil {
			_ = file.Close()
		}
	}
	base = nil
	once = sync.Once{}
}

func resetLoggerForTest(t *testing.T) {
	t.Helper()
	resetGlobalLoggerState()
	t.Cleanup(resetGlobalLoggerState)
}

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
	resetLoggerForTest(t)

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

func TestSetSink_CapturesLogMessage(t *testing.T) {
	resetLoggerForTest(t)
	t.Setenv("PAUSE_LOG_LEVEL", "debug")

	var gotLevel Level
	var gotMessage string
	SetSink(func(level Level, message string) {
		gotLevel = level
		gotMessage = message
	})

	Warnf("hello %s", "sink")

	if gotLevel != LevelWarn {
		t.Fatalf("sink level=%v want=%v", gotLevel, LevelWarn)
	}
	if gotMessage != "hello sink" {
		t.Fatalf("sink message=%q want=%q", gotMessage, "hello sink")
	}
}

func TestResetGlobalLoggerState_AllowsFreshInitialization(t *testing.T) {
	resetLoggerForTest(t)
	t.Setenv("PAUSE_LOG_LEVEL", "debug")

	var firstCalls int
	SetSink(func(level Level, message string) {
		firstCalls++
	})
	Infof("first init")
	if firstCalls != 1 {
		t.Fatalf("first sink calls=%d want=1", firstCalls)
	}

	resetGlobalLoggerState()
	t.Setenv("PAUSE_LOG_LEVEL", "error")

	var secondCalls int
	SetSink(func(level Level, message string) {
		secondCalls++
	})
	Infof("filtered after reset")
	Errorf("kept after reset")

	if secondCalls != 1 {
		t.Fatalf("second sink calls=%d want=1", secondCalls)
	}
}
