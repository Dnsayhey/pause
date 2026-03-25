//go:build windows

package windows

import (
	"errors"
	"testing"
)

func TestShowReminder_PrefersToast(t *testing.T) {
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
		if appID != "com.pause.app" || title != "Pause" || body != "Break started" {
			t.Fatalf("unexpected toast payload appID=%q title=%q body=%q", appID, title, body)
		}
		return nil
	}
	showBalloonNotification = func(_, _ string) error {
		calledBalloon = true
		return nil
	}

	n := windowsNotifier{appID: "com.pause.app"}
	if err := n.ShowReminder(" ", " "); err != nil {
		t.Fatalf("ShowReminder() err=%v", err)
	}
	if !calledToast || calledBalloon {
		t.Fatalf("toast/balloon path mismatch toast=%t balloon=%t", calledToast, calledBalloon)
	}
}

func TestShowReminder_FallsBackToBalloonError(t *testing.T) {
	origToast := showToastReminder
	origBalloon := showBalloonNotification
	t.Cleanup(func() {
		showToastReminder = origToast
		showBalloonNotification = origBalloon
	})

	toastErr := errors.New("toast failed")
	balloonErr := errors.New("balloon failed")
	showToastReminder = func(_, _, _ string) error { return toastErr }
	showBalloonNotification = func(_, _ string) error { return balloonErr }

	n := windowsNotifier{appID: "com.pause.app"}
	if err := n.ShowReminder("t", "b"); !errors.Is(err, balloonErr) {
		t.Fatalf("expected balloon error, got=%v", err)
	}
}
