//go:build windows

package windows

import (
	"errors"
	"testing"
)

func TestShowReminderPrefersToastPath(t *testing.T) {
	origToast := showToastReminder
	origBalloon := showBalloonNotification
	t.Cleanup(func() {
		showToastReminder = origToast
		showBalloonNotification = origBalloon
	})

	calledToast := false
	calledBalloon := false
	showToastReminder = func(appID, title, body string) error {
		calledToast = true
		if appID != "com.pause.app" {
			t.Fatalf("unexpected appID: %s", appID)
		}
		if title != "Pause" {
			t.Fatalf("unexpected title: %s", title)
		}
		if body != "Break started" {
			t.Fatalf("unexpected body: %s", body)
		}
		return nil
	}
	showBalloonNotification = func(_, _ string) error {
		calledBalloon = true
		return nil
	}

	n := windowsNotifier{appID: "com.pause.app"}
	if err := n.ShowReminder(" ", " "); err != nil {
		t.Fatalf("ShowReminder() error = %v", err)
	}
	if !calledToast {
		t.Fatalf("expected toast path to be called")
	}
	if calledBalloon {
		t.Fatalf("did not expect balloon fallback when toast succeeds")
	}
}

func TestShowReminderFallsBackWhenToastFails(t *testing.T) {
	origToast := showToastReminder
	origBalloon := showBalloonNotification
	t.Cleanup(func() {
		showToastReminder = origToast
		showBalloonNotification = origBalloon
	})

	toastErr := errors.New("toast failed")
	balloonErr := errors.New("balloon failed")

	showToastReminder = func(_, _, _ string) error {
		return toastErr
	}
	showBalloonNotification = func(_, _ string) error {
		return balloonErr
	}

	n := windowsNotifier{appID: "com.pause.app"}
	err := n.ShowReminder("t", "b")
	if !errors.Is(err, balloonErr) {
		t.Fatalf("expected balloon error, got %v", err)
	}
}
