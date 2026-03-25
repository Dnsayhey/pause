//go:build darwin

package darwin

import (
	"testing"

	"pause/internal/meta"
)

func TestIdleSecondsFromNanoseconds(t *testing.T) {
	if got := idleSecondsFromNanoseconds(0); got != 0 {
		t.Fatalf("got=%d want=0", got)
	}
	if got := idleSecondsFromNanoseconds(301_999_999_999); got != 301 {
		t.Fatalf("got=%d want=301", got)
	}
}

func TestValidateStartupExecutablePath(t *testing.T) {
	cases := []struct {
		path    string
		wantErr bool
	}{
		{path: "/Applications/Pause.app/Contents/MacOS/Pause", wantErr: false},
		{path: "/Volumes/Pause/Pause.app/Contents/MacOS/Pause", wantErr: true},
		{path: "/private/var/folders/xx/AppTranslocation/ABC/Pause.app/Contents/MacOS/Pause", wantErr: true},
	}
	for _, tc := range cases {
		err := validateStartupExecutablePath(tc.path)
		if (err != nil) != tc.wantErr {
			t.Fatalf("path=%q err=%v wantErr=%t", tc.path, err, tc.wantErr)
		}
	}
}

func TestNewDarwinAdapters_BundleIDResolution(t *testing.T) {
	adapters := NewAdapters("com.pause.app")
	manager, ok := adapters.StartupManager.(darwinStartupManager)
	if !ok {
		t.Fatalf("unexpected startup manager type")
	}
	if manager.helperBundleID != "com.pause.app.loginhelper" {
		t.Fatalf("helper bundle id mismatch: %q", manager.helperBundleID)
	}

	prev := meta.AppBundleID
	meta.AppBundleID = "com.pause.test"
	t.Cleanup(func() { meta.AppBundleID = prev })

	adapters = NewAdapters("")
	manager, ok = adapters.StartupManager.(darwinStartupManager)
	if !ok {
		t.Fatalf("unexpected startup manager type")
	}
	if manager.appID != "com.pause.test" || manager.helperBundleID != "com.pause.test.loginhelper" {
		t.Fatalf("default bundle id mismatch: appID=%q helper=%q", manager.appID, manager.helperBundleID)
	}
}
