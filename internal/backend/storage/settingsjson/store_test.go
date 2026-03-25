package settingsjson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	settingsdomain "pause/internal/backend/domain/settings"
)

func TestStoreCreatesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	got := store.Get()
	if !got.Enforcement.OverlaySkipAllowed {
		t.Fatalf("expected overlay skip allowed by default")
	}
	if got.UI.Theme != settingsdomain.UIThemeAuto {
		t.Fatalf("expected default theme %q, got %q", settingsdomain.UIThemeAuto, got.UI.Theme)
	}
	if !store.WasCreated() {
		t.Fatalf("expected WasCreated()=true when config file is newly created")
	}
}

func TestStoreUpdatePersistsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	off := false
	_, err = store.Update(settingsdomain.SettingsPatch{
		GlobalEnabled: &off,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore(reload) error = %v", err)
	}
	if reloaded.WasCreated() {
		t.Fatalf("expected WasCreated()=false for existing config file")
	}
	got := reloaded.Get()
	if got.GlobalEnabled {
		t.Fatalf("expected GlobalEnabled=false after reload")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "\"globalEnabled\": false") {
		t.Fatalf("expected settings file to persist updated value, got %s", string(raw))
	}
}

func TestStoreRecoversFromCorruptedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}

	got := store.Get()
	want := settingsdomain.DefaultSettings()
	if got.UI.Theme != want.UI.Theme {
		t.Fatalf("expected default theme %q after recovery, got %q", want.UI.Theme, got.UI.Theme)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "settings.json.corrupt.*.bak"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 backup file, got %d", len(matches))
	}
}
