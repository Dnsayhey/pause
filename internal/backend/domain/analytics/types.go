package analytics

type ReminderStat struct {
	ReminderID          int64   `json:"reminderId"`
	ReminderName        string  `json:"reminderName"`
	Enabled             bool    `json:"enabled"`
	ReminderType        string  `json:"reminderType"`
	TriggeredCount      int     `json:"triggeredCount"`
	CompletedCount      int     `json:"completedCount"`
	SkippedCount        int     `json:"skippedCount"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type SummaryStats struct {
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type WeeklyStats struct {
	FromSec   int64          `json:"fromSec"`
	ToSec     int64          `json:"toSec"`
	Reminders []ReminderStat `json:"reminders"`
	Summary   SummaryStats   `json:"summary"`
}

type Summary struct {
	FromSec             int64   `json:"fromSec"`
	ToSec               int64   `json:"toSec"`
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	CompletionRate      float64 `json:"completionRate"`
	SkipRate            float64 `json:"skipRate"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type TrendPoint struct {
	Day                 string  `json:"day"`
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	CompletionRate      float64 `json:"completionRate"`
	SkipRate            float64 `json:"skipRate"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type Trend struct {
	FromSec int64        `json:"fromSec"`
	ToSec   int64        `json:"toSec"`
	Points  []TrendPoint `json:"points"`
}

type BreakTypeDistributionItem struct {
	ReminderID      int64   `json:"reminderId"`
	ReminderName    string  `json:"reminderName"`
	TriggeredCount  int     `json:"triggeredCount"`
	CompletedCount  int     `json:"completedCount"`
	SkippedCount    int     `json:"skippedCount"`
	CompletionRate  float64 `json:"completionRate"`
	SkipRate        float64 `json:"skipRate"`
	TriggeredShare  float64 `json:"triggeredShare"`
	ReminderType    string  `json:"reminderType,omitempty"`
	ReminderEnabled bool    `json:"reminderEnabled"`
}

type BreakTypeDistribution struct {
	FromSec        int64                       `json:"fromSec"`
	ToSec          int64                       `json:"toSec"`
	TotalTriggered int                         `json:"totalTriggered"`
	Items          []BreakTypeDistributionItem `json:"items"`
}
