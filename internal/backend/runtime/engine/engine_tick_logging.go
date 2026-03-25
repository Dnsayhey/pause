package engine

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"pause/internal/backend/domain/reminder"
	"pause/internal/backend/domain/settings"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/logx"
)

func (e *Engine) logTickLocked(now time.Time, settings settings.Settings, reminders []reminder.Reminder, reason string, rawDeltaSec, appliedDeltaSec int, evt *scheduler.Event) {
	sessionStatus := "none"
	if view := e.session.CurrentView(now); view != nil {
		sessionStatus = view.Status
	}
	nextByID := e.scheduler.NextByID(reminders)
	nextSummary := "none"
	if len(nextByID) > 0 {
		parts := make([]string, 0, len(nextByID))
		for id, next := range nextByID {
			parts = append(parts, strconv.FormatInt(id, 10)+"="+strconv.Itoa(next))
		}
		sort.Strings(parts)
		nextSummary = strings.Join(parts, ",")
	}

	evtReasons := ""
	evtBreak := 0
	if evt != nil {
		evtBreak = evt.BreakSec
		reasons := make([]string, 0, len(evt.Reasons))
		for _, r := range evt.Reasons {
			reasons = append(reasons, strconv.FormatInt(int64(r), 10))
		}
		evtReasons = strings.Join(reasons, "+")
	}

	logx.Debugf(
		"engine.tick reason=%s now_unix=%d raw_delta=%d applied_delta=%d idle_sec=%d tick_active=%t session=%s next=%s evt_reasons=%s evt_break=%d",
		reason,
		now.Unix(),
		rawDeltaSec,
		appliedDeltaSec,
		e.currentIdleSec,
		e.lastTickActive,
		sessionStatus,
		nextSummary,
		evtReasons,
		evtBreak,
	)
}
