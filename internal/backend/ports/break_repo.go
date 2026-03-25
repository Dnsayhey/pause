package ports

import (
	"context"
	"time"
)

type BreakRepository interface {
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
