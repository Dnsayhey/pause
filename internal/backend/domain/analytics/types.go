package analytics

type ReminderStat struct {
	ReminderID          int64
	ReminderName        string
	Enabled             bool
	ReminderType        string
	TriggeredCount      int
	CompletedCount      int
	SkippedCount        int
	CanceledCount       int
	TotalActualBreakSec int
	AvgActualBreakSec   float64
}

type SummaryStats struct {
	TotalSessions       int
	TotalCompleted      int
	TotalSkipped        int
	TotalCanceled       int
	TotalActualBreakSec int
	AvgActualBreakSec   float64
}

type WeeklyStats struct {
	FromSec   int64
	ToSec     int64
	Reminders []ReminderStat
	Summary   SummaryStats
}

type Summary struct {
	FromSec             int64
	ToSec               int64
	TotalSessions       int
	TotalCompleted      int
	TotalSkipped        int
	TotalCanceled       int
	CompletionRate      float64
	SkipRate            float64
	TotalActualBreakSec int
	AvgActualBreakSec   float64
}

type TrendPoint struct {
	Day                 string
	TotalSessions       int
	TotalCompleted      int
	TotalSkipped        int
	TotalCanceled       int
	CompletionRate      float64
	SkipRate            float64
	TotalActualBreakSec int
	AvgActualBreakSec   float64
}

type Trend struct {
	FromSec int64
	ToSec   int64
	Points  []TrendPoint
}

type BreakTypeDistributionItem struct {
	ReminderID      int64
	ReminderName    string
	TriggeredCount  int
	CompletedCount  int
	SkippedCount    int
	CanceledCount   int
	CompletionRate  float64
	SkipRate        float64
	TriggeredShare  float64
	ReminderType    string
	ReminderEnabled bool
}

type BreakTypeDistribution struct {
	FromSec        int64
	ToSec          int64
	TotalTriggered int
	Items          []BreakTypeDistributionItem
}
