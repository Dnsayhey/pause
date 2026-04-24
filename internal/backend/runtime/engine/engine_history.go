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

type historyWrite struct {
	ctx    context.Context
	input  ports.BreakRecordInput
	logKey string
}

func reminderIDsFromEvent(evt *scheduler.Event) []int64 {
	if evt == nil || len(evt.Reasons) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(evt.Reasons))
	for _, reason := range evt.Reasons {
		ids = append(ids, int64(reason))
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

func (e *Engine) prepareBreakCompletedWriteLocked(view *state.BreakSessionView) *historyWrite {
	if e.history == nil || view == nil || e.activeHistoryBreak == nil {
		e.activeHistoryBreak = nil
		return nil
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
	e.activeHistoryBreak = nil
	return &historyWrite{
		ctx: ctx,
		input: ports.BreakRecordInput{
			StartedAt:       record.StartedAt,
			EndedAt:         view.EndsAt,
			Source:          record.Source,
			PlannedBreakSec: record.PlannedBreakSec,
			ActualBreakSec:  actualBreakSec,
			Skipped:         false,
			ReminderIDs:     record.ReminderIDs,
		},
		logKey: "history.break_complete_err",
	}
}

func (e *Engine) prepareBreakSkippedWriteLocked(now time.Time, view *state.BreakSessionView) *historyWrite {
	if e.history == nil || view == nil || e.activeHistoryBreak == nil {
		e.activeHistoryBreak = nil
		return nil
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
	e.activeHistoryBreak = nil
	return &historyWrite{
		ctx: ctx,
		input: ports.BreakRecordInput{
			StartedAt:       record.StartedAt,
			EndedAt:         now,
			Source:          record.Source,
			PlannedBreakSec: record.PlannedBreakSec,
			ActualBreakSec:  actualBreakSec,
			Skipped:         true,
			ReminderIDs:     record.ReminderIDs,
		},
		logKey: "history.break_skip_err",
	}
}

func (e *Engine) commitHistoryWrite(write *historyWrite) {
	if e == nil || e.history == nil || write == nil {
		return
	}
	if err := e.history.RecordBreak(write.ctx, write.input); err != nil {
		logx.Warnf("%s source=%s err=%v", write.logKey, write.input.Source, err)
	}
}
