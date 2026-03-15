package history

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	deliveryTypeOverlay      = "overlay"
	deliveryTypeNotification = "notification"
	sourceScheduled          = "scheduled"
	sourceManual             = "manual"
	statusRunning            = "running"
	statusCompleted          = "completed"
	statusSkipped            = "skipped"
	statusCanceled           = "canceled"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

type ReminderDefinition struct {
	ID           string
	Name         string
	Enabled      bool
	IntervalSec  int
	BreakSec     int
	DeliveryType string
}

type ReminderMutation struct {
	ID           string
	Name         *string
	Enabled      *bool
	IntervalSec  *int
	BreakSec     *int
	DeliveryType *string
}

type ReminderWeeklyStat struct {
	ReminderID          string  `json:"reminderId"`
	ReminderName        string  `json:"reminderName"`
	Enabled             bool    `json:"enabled"`
	DeliveryType        string  `json:"deliveryType"`
	TriggeredCount      int     `json:"triggeredCount"`
	CompletedCount      int     `json:"completedCount"`
	SkippedCount        int     `json:"skippedCount"`
	CanceledCount       int     `json:"canceledCount"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type WeeklySummary struct {
	TotalSessions       int     `json:"totalSessions"`
	TotalCompleted      int     `json:"totalCompleted"`
	TotalSkipped        int     `json:"totalSkipped"`
	TotalCanceled       int     `json:"totalCanceled"`
	TotalActualBreakSec int     `json:"totalActualBreakSec"`
	AvgActualBreakSec   float64 `json:"avgActualBreakSec"`
}

type WeeklyStats struct {
	WeekStartSec int64                `json:"weekStartSec"`
	WeekEndSec   int64                `json:"weekEndSec"`
	Reminders    []ReminderWeeklyStat `json:"reminders"`
	Summary      WeeklySummary        `json:"summary"`
}

func OpenStore(path string) (*Store, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return nil, errors.New("history db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", clean)
	if err != nil {
		return nil, err
	}
	// Keep SQLite usage simple and deterministic for a local desktop app.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("history migrate failed: %w", err)
	}
	return nil
}

func normalizeReminderID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func normalizeDeliveryType(deliveryType string) string {
	switch strings.ToLower(strings.TrimSpace(deliveryType)) {
	case deliveryTypeNotification:
		return deliveryTypeNotification
	default:
		return deliveryTypeOverlay
	}
}

func normalizeSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case sourceManual:
		return sourceManual
	default:
		return sourceScheduled
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizePositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func (s *Store) SyncReminders(reminders []ReminderDefinition) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	if len(reminders) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, reminder := range reminders {
		id := normalizeReminderID(reminder.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(reminder.Name)
		if name == "" {
			name = id
		}
		intervalSec := normalizePositive(reminder.IntervalSec, 60)
		breakSec := normalizePositive(reminder.BreakSec, 5)
		deliveryType := normalizeDeliveryType(reminder.DeliveryType)
		_, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO reminders(id, name, enabled, interval_sec, break_sec, delivery_type)
			 VALUES(?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   name=excluded.name,
			   enabled=excluded.enabled,
			   interval_sec=excluded.interval_sec,
			   break_sec=excluded.break_sec,
			   delivery_type=excluded.delivery_type,
			   updated_at=unixepoch()`,
			id,
			name,
			boolToInt(reminder.Enabled),
			intervalSec,
			breakSec,
			deliveryType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) ListReminders() ([]ReminderDefinition, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("history store is not initialized")
	}

	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, name, enabled, interval_sec, break_sec, delivery_type
		 FROM reminders
		 ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ReminderDefinition{}
	for rows.Next() {
		var r ReminderDefinition
		var enabledInt int
		if err := rows.Scan(&r.ID, &r.Name, &enabledInt, &r.IntervalSec, &r.BreakSec, &r.DeliveryType); err != nil {
			return nil, err
		}
		r.Enabled = enabledInt == 1
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) UpdateReminders(mutations []ReminderMutation) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	if len(mutations) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, mutation := range mutations {
		id := normalizeReminderID(mutation.ID)
		if id == "" {
			continue
		}

		current := ReminderDefinition{
			ID:           id,
			Name:         id,
			Enabled:      true,
			IntervalSec:  20 * 60,
			BreakSec:     20,
			DeliveryType: deliveryTypeOverlay,
		}

		row := tx.QueryRowContext(
			context.Background(),
			`SELECT name, enabled, interval_sec, break_sec, delivery_type
			 FROM reminders WHERE id = ?`,
			id,
		)
		var enabledInt int
		switch err := row.Scan(&current.Name, &enabledInt, &current.IntervalSec, &current.BreakSec, &current.DeliveryType); {
		case err == nil:
			current.Enabled = enabledInt == 1
		case errors.Is(err, sql.ErrNoRows):
		default:
			return err
		}

		if mutation.Name != nil {
			name := strings.TrimSpace(*mutation.Name)
			if name != "" {
				current.Name = name
			}
		}
		if mutation.Enabled != nil {
			current.Enabled = *mutation.Enabled
		}
		if mutation.IntervalSec != nil {
			current.IntervalSec = normalizePositive(*mutation.IntervalSec, current.IntervalSec)
		}
		if mutation.BreakSec != nil {
			current.BreakSec = normalizePositive(*mutation.BreakSec, current.BreakSec)
		}
		if mutation.DeliveryType != nil {
			current.DeliveryType = normalizeDeliveryType(*mutation.DeliveryType)
		}

		_, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO reminders(id, name, enabled, interval_sec, break_sec, delivery_type)
			 VALUES(?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   name=excluded.name,
			   enabled=excluded.enabled,
			   interval_sec=excluded.interval_sec,
			   break_sec=excluded.break_sec,
			   delivery_type=excluded.delivery_type,
			   updated_at=unixepoch()`,
			current.ID,
			current.Name,
			boolToInt(current.Enabled),
			current.IntervalSec,
			current.BreakSec,
			current.DeliveryType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func dedupeReminderIDs(reminderIDs []string) []string {
	if len(reminderIDs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(reminderIDs))
	for _, raw := range reminderIDs {
		id := normalizeReminderID(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func (s *Store) StartBreak(sessionID string, startedAt time.Time, source string, plannedBreakSec int, reminderIDs []string) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return errors.New("session id is required")
	}
	plannedSec := normalizePositive(plannedBreakSec, 1)
	reasons := dedupeReminderIDs(reminderIDs)

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO break_sessions(id, trigger_source, status, started_at, planned_break_sec, actual_break_sec)
		 VALUES(?, ?, ?, ?, ?, 0)`,
		id,
		normalizeSource(source),
		statusRunning,
		startedAt.UTC().Unix(),
		plannedSec,
	)
	if err != nil {
		return err
	}

	for _, reminderID := range reasons {
		name := reminderID
		intervalSec := plannedSec
		breakSec := plannedSec
		deliveryType := deliveryTypeOverlay

		row := tx.QueryRowContext(
			context.Background(),
			`SELECT name, interval_sec, break_sec, delivery_type
			 FROM reminders
			 WHERE id = ?`,
			reminderID,
		)
		switch err := row.Scan(&name, &intervalSec, &breakSec, &deliveryType); {
		case err == nil:
		case errors.Is(err, sql.ErrNoRows):
			_, err = tx.ExecContext(
				context.Background(),
				`INSERT OR IGNORE INTO reminders(id, name, enabled, interval_sec, break_sec, delivery_type)
				 VALUES(?, ?, 1, ?, ?, ?)`,
				reminderID,
				reminderID,
				plannedSec,
				plannedSec,
				deliveryTypeOverlay,
			)
			if err != nil {
				return err
			}
		default:
			return err
		}

		intervalSec = normalizePositive(intervalSec, plannedSec)
		breakSec = normalizePositive(breakSec, plannedSec)
		deliveryType = normalizeDeliveryType(deliveryType)
		if strings.TrimSpace(name) == "" {
			name = reminderID
		}

		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO break_session_reminders(
			   session_id, reminder_id, reminder_name_snapshot, interval_sec_snapshot, break_sec_snapshot, delivery_type_snapshot
			 ) VALUES(?, ?, ?, ?, ?, ?)`,
			id,
			reminderID,
			name,
			intervalSec,
			breakSec,
			deliveryType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) CompleteBreak(sessionID string, endedAt time.Time, actualBreakSec int) error {
	return s.finishBreak(sessionID, statusCompleted, endedAt, 0, actualBreakSec)
}

func (s *Store) SkipBreak(sessionID string, skippedAt time.Time, actualBreakSec int) error {
	return s.finishBreak(sessionID, statusSkipped, skippedAt, skippedAt.UTC().Unix(), actualBreakSec)
}

func (s *Store) finishBreak(sessionID string, status string, endedAt time.Time, skippedAtUnix int64, actualBreakSec int) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil
	}
	actualSec := actualBreakSec
	if actualSec < 0 {
		actualSec = 0
	}

	res, err := s.db.ExecContext(
		context.Background(),
		`UPDATE break_sessions
		 SET status = ?,
		     ended_at = ?,
		     actual_break_sec = ?,
		     skipped_at = ?,
		     updated_at = unixepoch()
		 WHERE id = ?
		   AND status = ?`,
		status,
		endedAt.UTC().Unix(),
		actualSec,
		nullIfZero(skippedAtUnix),
		id,
		statusRunning,
	)
	if err != nil {
		return err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil
	}
	return nil
}

func nullIfZero(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func (s *Store) QueryWeeklyStats(weekStart time.Time, weekEnd time.Time) (WeeklyStats, error) {
	if s == nil || s.db == nil {
		return WeeklyStats{}, errors.New("history store is not initialized")
	}
	startUnix := weekStart.UTC().Unix()
	endUnix := weekEnd.UTC().Unix()
	if endUnix <= startUnix {
		return WeeklyStats{}, errors.New("invalid week range")
	}

	rows, err := s.db.QueryContext(
		context.Background(),
		`WITH sessions_in_week AS (
		   SELECT id, status, actual_break_sec
		   FROM break_sessions
		   WHERE started_at >= ?
		     AND started_at < ?
		     AND status <> 'running'
		 )
		 SELECT
		   r.id,
		   r.name,
		   r.enabled,
		   r.delivery_type,
		   COUNT(s.id) AS triggered_count,
		   SUM(CASE WHEN s.status = 'completed' THEN 1 ELSE 0 END) AS completed_count,
		   SUM(CASE WHEN s.status = 'skipped' THEN 1 ELSE 0 END) AS skipped_count,
		   SUM(CASE WHEN s.status = 'canceled' THEN 1 ELSE 0 END) AS canceled_count,
		   COALESCE(SUM(CASE WHEN s.status = 'completed' THEN s.actual_break_sec ELSE 0 END), 0) AS total_actual_break_sec,
		   COALESCE(ROUND(AVG(CASE WHEN s.status = 'completed' THEN s.actual_break_sec END), 1), 0) AS avg_actual_break_sec
		 FROM reminders r
		 LEFT JOIN break_session_reminders bsr ON bsr.reminder_id = r.id
		 LEFT JOIN sessions_in_week s ON s.id = bsr.session_id
		 GROUP BY r.id, r.name, r.enabled, r.delivery_type
		 ORDER BY triggered_count DESC, r.name COLLATE NOCASE ASC`,
		startUnix,
		endUnix,
	)
	if err != nil {
		return WeeklyStats{}, err
	}
	defer rows.Close()

	stats := WeeklyStats{
		WeekStartSec: startUnix,
		WeekEndSec:   endUnix,
		Reminders:    []ReminderWeeklyStat{},
	}

	for rows.Next() {
		var row ReminderWeeklyStat
		var enabledInt int
		if err := rows.Scan(
			&row.ReminderID,
			&row.ReminderName,
			&enabledInt,
			&row.DeliveryType,
			&row.TriggeredCount,
			&row.CompletedCount,
			&row.SkippedCount,
			&row.CanceledCount,
			&row.TotalActualBreakSec,
			&row.AvgActualBreakSec,
		); err != nil {
			return WeeklyStats{}, err
		}
		row.Enabled = enabledInt == 1
		stats.Reminders = append(stats.Reminders, row)
	}
	if err := rows.Err(); err != nil {
		return WeeklyStats{}, err
	}

	err = s.db.QueryRowContext(
		context.Background(),
		`WITH sessions_in_week AS (
		   SELECT id, status, actual_break_sec
		   FROM break_sessions
		   WHERE started_at >= ?
		     AND started_at < ?
		     AND status <> 'running'
		 )
		 SELECT
		   COUNT(id) AS total_sessions,
		   COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS total_completed,
		   COALESCE(SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END), 0) AS total_skipped,
		   COALESCE(SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END), 0) AS total_canceled,
		   COALESCE(SUM(CASE WHEN status = 'completed' THEN actual_break_sec ELSE 0 END), 0) AS total_actual_break_sec,
		   COALESCE(ROUND(AVG(CASE WHEN status = 'completed' THEN actual_break_sec END), 1), 0) AS avg_actual_break_sec
		 FROM sessions_in_week`,
		startUnix,
		endUnix,
	).Scan(
		&stats.Summary.TotalSessions,
		&stats.Summary.TotalCompleted,
		&stats.Summary.TotalSkipped,
		&stats.Summary.TotalCanceled,
		&stats.Summary.TotalActualBreakSec,
		&stats.Summary.AvgActualBreakSec,
	)
	if err != nil {
		return WeeklyStats{}, err
	}

	return stats, nil
}
