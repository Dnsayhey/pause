package history

import (
	"context"
	"errors"
	"time"
)

type AnalyticsReminderStat struct {
	ReminderID          int64   `json:"reminderId"`
	ReminderName        string  `json:"reminderName"`
	Enabled             bool    `json:"enabled"`
	ReminderType        string  `json:"reminderType"`
	TriggeredCount      int     `json:"triggeredCount"`
	CompletedCount      int     `json:"completedCount"`
	SkippedCount        int     `json:"skippedCount"`
	CanceledCount       int     `json:"canceledCount"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsSummaryStats struct {
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsWeeklyStats struct {
	FromSec   int64                   `json:"fromSec"`
	ToSec     int64                   `json:"toSec"`
	Reminders []AnalyticsReminderStat `json:"reminders"`
	Summary   AnalyticsSummaryStats   `json:"summary"`
}

type AnalyticsSummary struct {
	FromSec             int64   `json:"fromSec"`
	ToSec               int64   `json:"toSec"`
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	CompletionRate      float64 `json:"completionRate"`
	SkipRate            float64 `json:"skipRate"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsTrendPoint struct {
	Day                 string  `json:"day"`
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	CompletionRate      float64 `json:"completionRate"`
	SkipRate            float64 `json:"skipRate"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type AnalyticsTrend struct {
	FromSec int64                 `json:"fromSec"`
	ToSec   int64                 `json:"toSec"`
	Points  []AnalyticsTrendPoint `json:"points"`
}

type AnalyticsBreakTypeDistributionItem struct {
	ReminderID      int64   `json:"reminderId"`
	ReminderName    string  `json:"reminderName"`
	TriggeredCount  int     `json:"triggeredCount"`
	CompletedCount  int     `json:"completedCount"`
	SkippedCount    int     `json:"skippedCount"`
	CanceledCount   int     `json:"canceledCount"`
	CompletionRate  float64 `json:"completionRate"`
	SkipRate        float64 `json:"skipRate"`
	TriggeredShare  float64 `json:"triggeredShare"`
	ReminderType    string  `json:"reminderType,omitempty"`
	ReminderEnabled bool    `json:"reminderEnabled"`
}

type AnalyticsBreakTypeDistribution struct {
	FromSec        int64                                `json:"fromSec"`
	ToSec          int64                                `json:"toSec"`
	TotalTriggered int                                  `json:"totalTriggered"`
	Items          []AnalyticsBreakTypeDistributionItem `json:"items"`
}

func (s *Store) QueryAnalyticsWeeklyStats(from time.Time, to time.Time) (AnalyticsWeeklyStats, error) {
	if s == nil || s.db == nil {
		return AnalyticsWeeklyStats{}, errors.New("history store is not initialized")
	}
	startUnix, endUnix, err := normalizeAnalyticsRange(from, to)
	if err != nil {
		return AnalyticsWeeklyStats{}, err
	}

	reminders, err := s.queryReminderAggregatesByRange(startUnix, endUnix)
	if err != nil {
		return AnalyticsWeeklyStats{}, err
	}
	summary, err := s.querySummaryAggregateByRange(startUnix, endUnix)
	if err != nil {
		return AnalyticsWeeklyStats{}, err
	}

	return AnalyticsWeeklyStats{
		FromSec:   startUnix,
		ToSec:     endUnix,
		Reminders: reminders,
		Summary:   summary,
	}, nil
}

func (s *Store) QueryAnalyticsSummary(from time.Time, to time.Time) (AnalyticsSummary, error) {
	if s == nil || s.db == nil {
		return AnalyticsSummary{}, errors.New("history store is not initialized")
	}
	startUnix, endUnix, err := normalizeAnalyticsRange(from, to)
	if err != nil {
		return AnalyticsSummary{}, err
	}

	summary, err := s.querySummaryAggregateByRange(startUnix, endUnix)
	if err != nil {
		return AnalyticsSummary{}, err
	}

	return AnalyticsSummary{
		FromSec:             startUnix,
		ToSec:               endUnix,
		TotalSessions:       summary.TotalSessions,
		TotalCompleted:      summary.TotalCompleted,
		TotalSkipped:        summary.TotalSkipped,
		TotalCanceled:       summary.TotalCanceled,
		CompletionRate:      ratio(summary.TotalCompleted, summary.TotalSessions),
		SkipRate:            ratio(summary.TotalSkipped, summary.TotalSessions),
		TotalActualBreakSec: summary.TotalActualBreakSec,
		AvgActualBreakSec:   summary.AvgActualBreakSec,
	}, nil
}

func (s *Store) QueryAnalyticsTrendByDay(from time.Time, to time.Time) (AnalyticsTrend, error) {
	if s == nil || s.db == nil {
		return AnalyticsTrend{}, errors.New("history store is not initialized")
	}
	startUnix, endUnix, err := normalizeAnalyticsRange(from, to)
	if err != nil {
		return AnalyticsTrend{}, err
	}

	rows, err := s.db.QueryContext(
		context.Background(),
		`WITH overlay_sessions AS (
		   SELECT DISTINCT session_id
		   FROM break_session_reminders
		   WHERE reminder_type_snapshot = 'rest'
		 ),
		 sessions AS (
		   SELECT bs.started_at, bs.status, bs.actual_break_sec
		   FROM break_sessions bs
		   INNER JOIN overlay_sessions os ON os.session_id = bs.id
		   WHERE bs.started_at >= ?
		     AND bs.started_at < ?
		 )
		 SELECT
		   strftime('%Y-%m-%d', started_at, 'unixepoch', 'localtime') AS day,
		   COUNT(*) AS total_sessions,
		   COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS total_completed,
		   COALESCE(SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END), 0) AS total_skipped,
		   COALESCE(SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END), 0) AS total_canceled,
		   COALESCE(SUM(CASE WHEN status = 'completed' THEN actual_break_sec ELSE 0 END), 0) AS total_actual_break_sec,
		   COALESCE(ROUND(AVG(CASE WHEN status = 'completed' THEN actual_break_sec END), 1), 0) AS avg_actual_break_sec
		 FROM sessions
		 GROUP BY day
		 ORDER BY day ASC`,
		startUnix,
		endUnix,
	)
	if err != nil {
		return AnalyticsTrend{}, err
	}
	defer rows.Close()

	result := AnalyticsTrend{
		FromSec: startUnix,
		ToSec:   endUnix,
		Points:  []AnalyticsTrendPoint{},
	}

	for rows.Next() {
		row := AnalyticsTrendPoint{}
		if err := rows.Scan(
			&row.Day,
			&row.TotalSessions,
			&row.TotalCompleted,
			&row.TotalSkipped,
			&row.TotalCanceled,
			&row.TotalActualBreakSec,
			&row.AvgActualBreakSec,
		); err != nil {
			return AnalyticsTrend{}, err
		}
		row.CompletionRate = ratio(row.TotalCompleted, row.TotalSessions)
		row.SkipRate = ratio(row.TotalSkipped, row.TotalSessions)
		result.Points = append(result.Points, row)
	}
	if err := rows.Err(); err != nil {
		return AnalyticsTrend{}, err
	}
	return result, nil
}

func (s *Store) QueryAnalyticsBreakTypeDistribution(from time.Time, to time.Time) (AnalyticsBreakTypeDistribution, error) {
	if s == nil || s.db == nil {
		return AnalyticsBreakTypeDistribution{}, errors.New("history store is not initialized")
	}
	startUnix, endUnix, err := normalizeAnalyticsRange(from, to)
	if err != nil {
		return AnalyticsBreakTypeDistribution{}, err
	}

	aggregates, err := s.queryReminderAggregatesByRange(startUnix, endUnix)
	if err != nil {
		return AnalyticsBreakTypeDistribution{}, err
	}

	result := AnalyticsBreakTypeDistribution{
		FromSec: startUnix,
		ToSec:   endUnix,
		Items:   []AnalyticsBreakTypeDistributionItem{},
	}
	for _, row := range aggregates {
		item := AnalyticsBreakTypeDistributionItem{
			ReminderID:      row.ReminderID,
			ReminderName:    row.ReminderName,
			TriggeredCount:  row.TriggeredCount,
			CompletedCount:  row.CompletedCount,
			SkippedCount:    row.SkippedCount,
			CanceledCount:   row.CanceledCount,
			CompletionRate:  ratio(row.CompletedCount, row.TriggeredCount),
			SkipRate:        ratio(row.SkippedCount, row.TriggeredCount),
			ReminderType:    row.ReminderType,
			ReminderEnabled: row.Enabled,
		}
		result.TotalTriggered += item.TriggeredCount
		result.Items = append(result.Items, item)
	}
	for i := range result.Items {
		result.Items[i].TriggeredShare = ratio(result.Items[i].TriggeredCount, result.TotalTriggered)
	}
	return result, nil
}

func (s *Store) queryReminderAggregatesByRange(startUnix int64, endUnix int64) ([]AnalyticsReminderStat, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`WITH sessions_in_range AS (
		   SELECT id, status, actual_break_sec
		   FROM break_sessions
		   WHERE started_at >= ?
		     AND started_at < ?
		 ),
		 history_agg AS (
		   SELECT
		     bsr.reminder_id AS reminder_id,
		     bsr.reminder_name_snapshot AS reminder_name,
		     bsr.reminder_type_snapshot AS reminder_type,
		     COUNT(s.id) AS triggered_count,
		     COALESCE(SUM(CASE WHEN s.status = 'completed' THEN 1 ELSE 0 END), 0) AS completed_count,
		     COALESCE(SUM(CASE WHEN s.status = 'skipped' THEN 1 ELSE 0 END), 0) AS skipped_count,
		     COALESCE(SUM(CASE WHEN s.status = 'canceled' THEN 1 ELSE 0 END), 0) AS canceled_count,
		     COALESCE(SUM(CASE WHEN s.status = 'completed' THEN s.actual_break_sec ELSE 0 END), 0) AS total_actual_break_sec,
		     COALESCE(ROUND(AVG(CASE WHEN s.status = 'completed' THEN s.actual_break_sec END), 1), 0) AS avg_actual_break_sec
		   FROM sessions_in_range s
		   INNER JOIN break_session_reminders bsr ON bsr.session_id = s.id
		   WHERE bsr.reminder_type_snapshot = 'rest'
		   GROUP BY
		     bsr.reminder_id,
		     bsr.reminder_name_snapshot,
		     bsr.reminder_type_snapshot
		 ),
		 active_zero AS (
		   SELECT
		     r.id AS reminder_id,
		     r.name AS reminder_name,
		     r.reminder_type AS reminder_type,
		     0 AS triggered_count,
		     0 AS completed_count,
		     0 AS skipped_count,
		     0 AS canceled_count,
		     0 AS total_actual_break_sec,
		     0.0 AS avg_actual_break_sec
		   FROM reminders r
		   WHERE r.deleted_at IS NULL
		     AND r.reminder_type = 'rest'
		     AND NOT EXISTS (
		       SELECT 1
		       FROM history_agg h
		       WHERE h.reminder_id = r.id
		     )
		 ),
		 combined AS (
		   SELECT * FROM history_agg
		   UNION ALL
		   SELECT * FROM active_zero
		 )
		 SELECT
		   c.reminder_id,
		   c.reminder_name,
		   COALESCE(r.enabled, 0) AS enabled,
		   c.reminder_type,
		   c.triggered_count,
		   c.completed_count,
		   c.skipped_count,
		   c.canceled_count,
		   c.total_actual_break_sec,
		   c.avg_actual_break_sec
		 FROM combined c
		 LEFT JOIN reminders r ON r.id = c.reminder_id AND r.deleted_at IS NULL
		 ORDER BY c.triggered_count DESC, c.reminder_name COLLATE NOCASE ASC`,
		startUnix,
		endUnix,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []AnalyticsReminderStat{}
	for rows.Next() {
		row := AnalyticsReminderStat{}
		var enabledInt int
		if err := rows.Scan(
			&row.ReminderID,
			&row.ReminderName,
			&enabledInt,
			&row.ReminderType,
			&row.TriggeredCount,
			&row.CompletedCount,
			&row.SkippedCount,
			&row.CanceledCount,
			&row.TotalActualBreakSec,
			&row.AvgActualBreakSec,
		); err != nil {
			return nil, err
		}
		row.Enabled = enabledInt == 1
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) querySummaryAggregateByRange(startUnix int64, endUnix int64) (AnalyticsSummaryStats, error) {
	summary := AnalyticsSummaryStats{}
	err := s.db.QueryRowContext(
		context.Background(),
		`WITH overlay_sessions AS (
		   SELECT DISTINCT session_id
		   FROM break_session_reminders
		   WHERE reminder_type_snapshot = 'rest'
		 ),
		 sessions_in_range AS (
		   SELECT bs.id, bs.status, bs.actual_break_sec
		   FROM break_sessions bs
		   INNER JOIN overlay_sessions os ON os.session_id = bs.id
		   WHERE bs.started_at >= ?
		     AND bs.started_at < ?
		 )
		 SELECT
		   COUNT(id) AS total_sessions,
		   COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS total_completed,
		   COALESCE(SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END), 0) AS total_skipped,
		   COALESCE(SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END), 0) AS total_canceled,
		   COALESCE(SUM(CASE WHEN status = 'completed' THEN actual_break_sec ELSE 0 END), 0) AS total_actual_break_sec,
		   COALESCE(ROUND(AVG(CASE WHEN status = 'completed' THEN actual_break_sec END), 1), 0) AS avg_actual_break_sec
		 FROM sessions_in_range`,
		startUnix,
		endUnix,
	).Scan(
		&summary.TotalSessions,
		&summary.TotalCompleted,
		&summary.TotalSkipped,
		&summary.TotalCanceled,
		&summary.TotalActualBreakSec,
		&summary.AvgActualBreakSec,
	)
	if err != nil {
		return AnalyticsSummaryStats{}, err
	}
	return summary, nil
}

func normalizeAnalyticsRange(from time.Time, to time.Time) (int64, int64, error) {
	startUnix := from.UTC().Unix()
	endUnix := to.UTC().Unix()
	if endUnix <= startUnix {
		return 0, 0, errors.New("invalid time range")
	}
	return startUnix, endUnix, nil
}

func ratio(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
