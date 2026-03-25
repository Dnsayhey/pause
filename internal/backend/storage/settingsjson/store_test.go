package settingsjson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	settingsdomain "pause/internal/backend/domain/settings"
)

func TestOpenStore_NewFileCreatesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}

	got := store.Get()
	if !store.WasCreated() {
		t.Fatalf("expected newly created store")
	}
	if got.UI.Theme != settingsdomain.UIThemeAuto {
		t.Fatalf("default theme mismatch: got=%q", got.UI.Theme)
	}
	if !got.Enforcement.OverlaySkipAllowed {
		t.Fatalf("expected overlay skip allowed by default")
	}
}

func TestStore_UpdatePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}

	off := false
	if _, err := store.Update(settingsdomain.SettingsPatch{GlobalEnabled: &off}); err != nil {
		t.Fatalf("Update() err=%v", err)
	}

	reloaded, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore(reload) err=%v", err)
	}
	if reloaded.Get().GlobalEnabled {
		t.Fatalf("expected persisted globalEnabled=false")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() err=%v", err)
	}
	if !strings.Contains(string(raw), "\"globalEnabled\": false") {
		t.Fatalf("expected persisted json contains globalEnabled=false")
	}
}

func TestOpenStore_CorruptedConfigBackedUpAndRecovered(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("WriteFile() err=%v", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() err=%v", err)
	}
	if store.Get().UI.Theme != settingsdomain.DefaultSettings().UI.Theme {
		t.Fatalf("expected recovered defaults")
	}

	matches, err := filepath.Glob(filepath.Join(dir, "settings.json.corrupt.*.bak"))
	if err != nil {
		t.Fatalf("Glob() err=%v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("backup file count mismatch: got=%d", len(matches))
	}
}
