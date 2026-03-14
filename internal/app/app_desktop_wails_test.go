//go:build wails

package app

import (
	"testing"

	"pause/internal/core/config"
)

func TestSmoothCountdownValue_SnapsOnLargeDownwardJump(t *testing.T) {
	got := smoothCountdownValue(1200, 590, 1)
	if got != 590 {
		t.Fatalf("smoothCountdownValue() = %d, want %d", got, 590)
	}
}

func TestSmoothCountdownValue_SmoothOnSmallDownwardJump(t *testing.T) {
	got := smoothCountdownValue(100, 97, 1)
	if got != 98 {
		t.Fatalf("smoothCountdownValue() = %d, want %d", got, 98)
	}
}

func TestBuildSmoothKey_ChangesWhenRuleChanges(t *testing.T) {
	a := config.DefaultSettings()
	b := a
	b.Eye.IntervalSec = 30 * 60

	if buildSmoothKey(a) == buildSmoothKey(b) {
		t.Fatalf("buildSmoothKey should differ when eye interval changes")
	}
}
