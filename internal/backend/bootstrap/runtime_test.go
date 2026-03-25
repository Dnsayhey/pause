package bootstrap

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewRuntime_RejectsEmptyConfigPath(t *testing.T) {
	r, err := NewRuntime("   ", "com.example.pause")
	if err == nil {
		if r != nil {
			_ = r.Close()
		}
		t.Fatalf("expected empty config path error")
	}
}

func TestNewRuntime_BuildsCompleteRuntime(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "settings.json")
	r, err := NewRuntime(configPath, "com.example.pause")
	if err != nil {
		t.Fatalf("NewRuntime() err=%v", err)
	}
	defer r.Close()

	if r.Settings == nil || r.History == nil || r.Engine == nil {
		t.Fatalf("runtime core wiring is incomplete")
	}
	if r.ReminderService == nil || r.SettingsService == nil || r.AnalyticsService == nil {
		t.Fatalf("runtime services wiring is incomplete")
	}
	if r.HistoryPath == "" {
		t.Fatalf("expected history path")
	}

	if _, err := r.ReminderService.List(context.Background()); err != nil {
		t.Fatalf("ReminderService.List() err=%v", err)
	}
}
