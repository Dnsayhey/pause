package engine

import (
	"context"
	"sort"
	"time"

	"pause/internal/backend/ports"
	"pause/internal/backend/runtime/scheduler"
	"pause/internal/backend/runtime/state"
	"pause/internal/logx"
)

func reminderIDsFromEvent(evt *scheduler.Event) []int64 {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(evt.Reasons))
	for _, reason := range evt.Reasons {
		id := normalizeReminderID(int64(reason))
		if id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	uniq := ids[:0]
	var last int64
	lastSet := false
	for _, id := range ids {
		if lastSet && id == last {
			continue
		}
		uniq = append(uniq, id)
		last = id
		lastSet = true
	}
	return uniq
}

func (e *Engine) recordBreakStartedLocked(now time.Time, source string, evt *scheduler.Event) {
	e.activeHistoryBreak = nil
	if e.history == nil || evt == nil || evt.BreakSec <= 0 {
		return
	}

	e.activeHistoryBreak = &pendingHistoryBreak{
		StartedAt:       now,
		Source:          source,
		PlannedBreakSec: evt.BreakSec,
		ReminderIDs:     reminderIDsFromEvent(evt),
	}
}

func (e *Engine) recordBreakCompletedLocked(view *state.BreakSessionView) {
	if e.history == nil || view == nil || e.activeHistoryBreak == nil {
		e.activeHistoryBreak = nil
		return
	}
	actualBreakSec := int(view.EndsAt.Sub(view.StartedAt).Seconds())
	if actualBreakSec < 0 {
		actualBreakSec = 0
	}
	record := *e.activeHistoryBreak
	ctx := e.runCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.history.RecordBreak(ctx, ports.BreakRecordInput{
		StartedAt:       record.StartedAt,
		EndedAt:         view.EndsAt,
		Source:          record.Source,
		PlannedBreakSec: record.PlannedBreakSec,
		ActualBreakSec:  actualBreakSec,
		Skipped:         false,
		ReminderIDs:     record.ReminderIDs,
	}); err != nil {
		logx.Warnf("history.break_complete_err source=%s err=%v", record.Source, err)
	}
	e.activeHistoryBreak = nil
}

func (e *Engine) recordBreakSkippedLocked(now time.Time, view *state.BreakSessionView) {
	if e.history == nil || view == nil || e.activeHistoryBreak == nil {
		e.activeHistoryBreak = nil
		return
	}
	actualBreakSec := int(now.Sub(view.StartedAt).Seconds())
	if actualBreakSec < 0 {
		actualBreakSec = 0
	}
	record := *e.activeHistoryBreak
	ctx := e.runCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.history.RecordBreak(ctx, ports.BreakRecordInput{
		StartedAt:       record.StartedAt,
		EndedAt:         now,
		Source:          record.Source,
		PlannedBreakSec: record.PlannedBreakSec,
		ActualBreakSec:  actualBreakSec,
		Skipped:         true,
		ReminderIDs:     record.ReminderIDs,
	}); err != nil {
		logx.Warnf("history.break_skip_err source=%s err=%v", record.Source, err)
	}
	e.activeHistoryBreak = nil
}
