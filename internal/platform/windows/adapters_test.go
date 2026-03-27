//go:build windows

package windows

import (
	"errors"
	"testing"

	"pause/internal/backend/ports"
)

func TestShowReminder_PrefersToast(t *testing.T) {
	origToast := showToastReminder
	t.Cleanup(func() {
		showToastReminder = origToast
	})

	calledToast := false
	showToastReminder = func(appID, title, body string) error {
		calledToast = true
		if appID != "com.pause.app" || title != "Pause" || body != "Break started" {
			t.Fatalf("unexpected toast payload appID=%q title=%q body=%q", appID, title, body)
		}
		return nil
	}

	n := windowsNotifier{appID: "com.pause.app"}
	if err := n.ShowReminder(" ", " "); err != nil {
		t.Fatalf("ShowReminder() err=%v", err)
	}
	if !calledToast {
		t.Fatalf("expected toast sender to be called")
	}
}

func TestShowReminder_ReturnsToastError(t *testing.T) {
	origToast := showToastReminder
	t.Cleanup(func() {
		showToastReminder = origToast
	})

	toastErr := errors.New("toast failed")
	showToastReminder = func(_, _, _ string) error { return toastErr }

	n := windowsNotifier{appID: "com.pause.app"}
	if err := n.ShowReminder("t", "b"); !errors.Is(err, toastErr) {
		t.Fatalf("expected toast error, got=%v", err)
	}
}

func TestGetNotificationCapability_MapsNativeSetting(t *testing.T) {
	origQuery := queryToastSetting
	t.Cleanup(func() {
		queryToastSetting = origQuery
	})

	queryToastSetting = func(appID string) (string, error) {
		if appID != "com.pause.app" {
			t.Fatalf("unexpected appID=%q", appID)
		}
		return "DisabledForApplication", nil
	}

	got := (windowsNotifier{appID: "com.pause.app"}).GetNotificationCapability()
	if got.PermissionState != ports.NotificationPermissionDenied {
		t.Fatalf("permissionState=%q want=%q", got.PermissionState, ports.NotificationPermissionDenied)
	}
	if got.CanRequest {
		t.Fatalf("CanRequest=%t want=false", got.CanRequest)
	}
	if !got.CanOpenSettings {
		t.Fatalf("CanOpenSettings=%t want=true", got.CanOpenSettings)
	}
}

func TestOpenNotificationSettings_UsesNativeOpener(t *testing.T) {
	origOpen := openNotificationSettings
	t.Cleanup(func() {
		openNotificationSettings = origOpen
	})

	called := false
	openNotificationSettings = func() error {
		called = true
		return nil
	}

	if err := (windowsNotifier{}).OpenNotificationSettings(); err != nil {
		t.Fatalf("OpenNotificationSettings() err=%v", err)
	}
	if !called {
		t.Fatalf("expected native opener to be called")
	}
}

func TestBuildWindowsToastXML_EscapesText(t *testing.T) {
	got, err := buildWindowsToastXML(`A&B`, `<hello>"world"`)
	if err != nil {
		t.Fatalf("buildWindowsToastXML() err=%v", err)
	}
	want := "<toast><visual><binding template=\"ToastGeneric\"><text>A&amp;B</text><text>&lt;hello&gt;&#34;world&#34;</text></binding></visual></toast>"
	if got != want {
		t.Fatalf("toast xml=%q want=%q", got, want)
	}
}
