//go:build darwin

package darwin

import (
	"testing"

	"pause/internal/meta"
)

func TestIdleSecondsFromNanoseconds(t *testing.T) {
	if got := idleSecondsFromNanoseconds(0); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := idleSecondsFromNanoseconds(301_999_999_999); got != 301 {
		t.Fatalf("expected 301, got %d", got)
	}
}

func TestValidateStartupExecutablePath(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "normal app path",
			path:    "/Applications/Pause.app/Contents/MacOS/Pause",
			wantErr: false,
		},
		{
			name:    "mounted volume path",
			path:    "/Volumes/Pause/Pause.app/Contents/MacOS/Pause",
			wantErr: true,
		},
		{
			name:    "app translocation path",
			path:    "/private/var/folders/xx/AppTranslocation/ABC/Pause.app/Contents/MacOS/Pause",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		err := validateStartupExecutablePath(tc.path)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: expected no error, got %v", tc.name, err)
		}
	}
}

func TestNewDarwinAdaptersSetsHelperBundleID(t *testing.T) {
	adapters := NewAdapters("com.pause.app")
	manager, ok := adapters.StartupManager.(darwinStartupManager)
	if !ok {
		t.Fatalf("expected darwinStartupManager type")
	}
	if manager.appID != "com.pause.app" {
		t.Fatalf("unexpected appID: %q", manager.appID)
	}
	if manager.helperBundleID != "com.pause.app.loginhelper" {
		t.Fatalf("unexpected helper bundle id: %q", manager.helperBundleID)
	}
}

func TestNewDarwinAdaptersUsesDefaultBundleIDWhenInputEmpty(t *testing.T) {
	previous := meta.AppBundleID
	meta.AppBundleID = "com.pause.test"
	t.Cleanup(func() {
		meta.AppBundleID = previous
	})

	adapters := NewAdapters("")
	manager, ok := adapters.StartupManager.(darwinStartupManager)
	if !ok {
		t.Fatalf("expected darwinStartupManager type")
	}
	if manager.appID != "com.pause.test" {
		t.Fatalf("unexpected default appID: %q", manager.appID)
	}
	if manager.helperBundleID != "com.pause.test.loginhelper" {
		t.Fatalf("unexpected default helper bundle id: %q", manager.helperBundleID)
	}
}
