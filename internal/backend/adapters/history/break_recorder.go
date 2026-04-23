package historyadapter

import (
	"context"
	"time"

	"pause/internal/backend/ports"
)

type BreakRecorder struct {
	store HistoryStoreRef
}

type HistoryStoreRef interface {
	RecordBreak(ctx context.Context, startedAt time.Time, endedAt time.Time, source string, plannedBreakSec int, actualBreakSec int, skipped bool, reminderIDs []int64) error
}

var _ ports.BreakRecorder = (*BreakRecorder)(nil)

func NewBreakRecorder(store HistoryStoreRef) *BreakRecorder {
	return &BreakRecorder{store: store}
}

func (r *BreakRecorder) RecordBreak(ctx context.Context, input ports.BreakRecordInput) error {
	if r == nil || r.store == nil {
		return errHistoryStoreUnavailable
	}
	return r.store.RecordBreak(ctx, input.StartedAt, input.EndedAt, input.Source, input.PlannedBreakSec, input.ActualBreakSec, input.Skipped, input.ReminderIDs)
}
