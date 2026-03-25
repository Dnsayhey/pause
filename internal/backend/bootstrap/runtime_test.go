package bootstrap

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewRuntimeRejectsEmptyConfigPath(t *testing.T) {
	runtime, err := NewRuntime("   ", "com.example.pause")
	if err == nil {
		if runtime != nil {
			_ = runtime.Close()
		}
		t.Fatalf("expected empty config path to fail")
	}
}

func TestNewRuntimeBuildsAllWiring(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "settings.json")

	runtime, err := NewRuntime(configPath, "com.example.pause")
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer runtime.Close()

	if runtime.SettingsStore == nil {
		t.Fatalf("expected settings store to be initialized")
	}
	if runtime.History == nil {
		t.Fatalf("expected history store to be initialized")
	}
	if runtime.Engine == nil {
		t.Fatalf("expected engine to be initialized")
	}
	if runtime.ReminderService == nil {
		t.Fatalf("expected reminder service to be initialized")
	}
	if runtime.AnalyticsService == nil {
		t.Fatalf("expected analytics service to be initialized")
	}
	if runtime.SettingsService == nil {
		t.Fatalf("expected settings service to be initialized")
	}
	if runtime.HistoryPath == "" {
		t.Fatalf("expected history path to be set")
	}

	if _, err := runtime.ReminderService.List(context.Background()); err != nil {
		t.Fatalf("ReminderService.List() error = %v", err)
	}
}
