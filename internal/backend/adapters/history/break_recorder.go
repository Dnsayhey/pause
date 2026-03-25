package historyadapter

import (
	"context"
	"time"

	"pause/internal/core/service"
)

type BreakRecorder struct {
	store HistoryStoreRef
}

type HistoryStoreRef interface {
	RecordBreak(
		ctx context.Context,
		startedAt time.Time,
		endedAt time.Time,
		source string,
		plannedBreakSec int,
		actualBreakSec int,
		skipped bool,
		reminderIDs []int64,
	) error
}

var _ service.BreakHistoryRecorder = (*BreakRecorder)(nil)

func NewBreakRecorder(store HistoryStoreRef) *BreakRecorder {
	return &BreakRecorder{store: store}
}

func (r *BreakRecorder) RecordBreak(
	ctx context.Context,
	startedAt time.Time,
	endedAt time.Time,
	source string,
	plannedBreakSec int,
	actualBreakSec int,
	skipped bool,
	reminderIDs []int64,
) error {
	if r == nil || r.store == nil {
		return errHistoryStoreUnavailable
	}
	return r.store.RecordBreak(ctx, startedAt, endedAt, source, plannedBreakSec, actualBreakSec, skipped, reminderIDs)
}
