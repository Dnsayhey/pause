//go:build darwin && wails

package main

/*
 */
import "C"

import (
	"fmt"

	"pause/internal/diag"
)

const (
	windowDebugEventConfigure      = 1
	windowDebugEventShowActivate   = 2
	windowDebugEventShowNoActivate = 3
	windowDebugEventHide           = 4
	windowDebugEventDidBecomeKey   = 5
	windowDebugEventDidBecomeMain  = 6
	windowDebugEventDidDemini      = 7
	windowDebugEventDidMini        = 8
	windowDebugEventDidResignKey   = 9
	windowDebugEventDidResignMain  = 10
	windowDebugEventDidExpose      = 11
	windowDebugEventDidOcclusion   = 12
	windowDebugEventAppActive      = 13
	windowDebugEventAppResign      = 14
	windowDebugEventSnapshotChange = 15
)

func windowDebugEventName(eventID int) string {
	switch eventID {
	case windowDebugEventConfigure:
		return "configure"
	case windowDebugEventShowActivate:
		return "show_activate"
	case windowDebugEventShowNoActivate:
		return "show_no_activate"
	case windowDebugEventHide:
		return "hide"
	case windowDebugEventDidBecomeKey:
		return "did_become_key"
	case windowDebugEventDidBecomeMain:
		return "did_become_main"
	case windowDebugEventDidDemini:
		return "did_deminiaturize"
	case windowDebugEventDidMini:
		return "did_miniaturize"
	case windowDebugEventDidResignKey:
		return "did_resign_key"
	case windowDebugEventDidResignMain:
		return "did_resign_main"
	case windowDebugEventDidExpose:
		return "did_expose"
	case windowDebugEventDidOcclusion:
		return "did_change_occlusion_state"
	case windowDebugEventAppActive:
		return "app_did_become_active"
	case windowDebugEventAppResign:
		return "app_did_resign_active"
	case windowDebugEventSnapshotChange:
		return "snapshot_changed"
	default:
		return fmt.Sprintf("unknown_%d", eventID)
	}
}

//export windowDebugEventGo
func windowDebugEventGo(eventID C.int, visible C.int, key C.int, mainWindow C.int, popover C.int) {
	diag.Logf(
		"window.event event_id=%d event=%s visible=%t key=%t main=%t popover=%t",
		int(eventID),
		windowDebugEventName(int(eventID)),
		int(visible) == 1,
		int(key) == 1,
		int(mainWindow) == 1,
		int(popover) == 1,
	)
}
