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
	reminderTypeRest   = "rest"
	reminderTypeNotify = "notify"
	sourceScheduled    = "scheduled"
	sourceManual       = "manual"
	statusCompleted    = "completed"
	statusSkipped      = "skipped"
)

var (
	ErrReminderAlreadyExists = errors.New("reminder already exists")
	ErrReminderNotFound      = errors.New("reminder not found")
	errContextRequired       = errors.New("context is required")
	errStoreNotInitialized   = errors.New("history store is not initialized")
	errReminderIDRequired    = errors.New("reminder id is required")
)

//go:embed schema.sql
var schemaSQL string

type HistoryStore struct {
	db *sql.DB
}

type Reminder struct {
	ID           int64
	Name         string
	Enabled      bool
	IntervalSec  int
	BreakSec     int
	ReminderType string
}

type ReminderPatch struct {
	Name         *string
	Enabled      *bool
	IntervalSec  *int
	BreakSec     *int
	ReminderType *string
}

func OpenHistoryStore(ctx context.Context, path string) (*HistoryStore, error) {
	if ctx == nil {
		return nil, errContextRequired
	}

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

	store := &HistoryStore{db: db}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *HistoryStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *HistoryStore) migrate(ctx context.Context) error {
	if err := ensureStore(ctx, s); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("history migrate failed: %w", err)
	}
	return nil
}

func isValidReminderType(reminderType string) bool {
	return reminderType == reminderTypeRest || reminderType == reminderTypeNotify
}

func isValidSource(source string) bool {
	return source == sourceScheduled || source == sourceManual
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func validateReminderName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("reminder name is required")
	}
	if name != strings.TrimSpace(name) {
		return errors.New("reminder name cannot have leading or trailing spaces")
	}
	return nil
}

func ensureStore(ctx context.Context, s *HistoryStore) error {
	if ctx == nil {
		return errContextRequired
	}
	if s == nil || s.db == nil {
		return errStoreNotInitialized
	}
	return nil
}

func validateReminder(reminder Reminder) error {
	if err := validateReminderName(reminder.Name); err != nil {
		return err
	}
	if reminder.IntervalSec <= 0 {
		return errors.New("reminder intervalSec must be > 0")
	}
	if reminder.BreakSec <= 0 {
		return errors.New("reminder breakSec must be > 0")
	}
	if !isValidReminderType(reminder.ReminderType) {
		return errors.New("reminder reminderType must be rest or notify")
	}
	return nil
}

func applyReminderPatch(current Reminder, patch ReminderPatch) (Reminder, error) {
	if patch.Name != nil {
		if err := validateReminderName(*patch.Name); err != nil {
			return Reminder{}, err
		}
		current.Name = *patch.Name
	}
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.IntervalSec != nil {
		if *patch.IntervalSec <= 0 {
			return Reminder{}, errors.New("reminder intervalSec must be > 0")
		}
		current.IntervalSec = *patch.IntervalSec
	}
	if patch.BreakSec != nil {
		if *patch.BreakSec <= 0 {
			return Reminder{}, errors.New("reminder breakSec must be > 0")
		}
		current.BreakSec = *patch.BreakSec
	}
	if patch.ReminderType != nil {
		if !isValidReminderType(*patch.ReminderType) {
			return Reminder{}, errors.New("reminder reminderType must be rest or notify")
		}
		current.ReminderType = *patch.ReminderType
	}
	return current, nil
}

func (s *HistoryStore) ListReminders(ctx context.Context) ([]Reminder, error) {
	if err := ensureStore(ctx, s); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, enabled, interval_sec, break_sec, reminder_type
		 FROM reminders
		 WHERE deleted_at IS NULL
		 ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Reminder{}
	for rows.Next() {
		var r Reminder
		var enabledInt int
		if err := rows.Scan(&r.ID, &r.Name, &enabledInt, &r.IntervalSec, &r.BreakSec, &r.ReminderType); err != nil {
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

func (s *HistoryStore) UpdateReminder(ctx context.Context, reminderID int64, patch ReminderPatch) error {
	if err := ensureStore(ctx, s); err != nil {
		return err
	}
	if reminderID <= 0 {
		return errReminderIDRequired
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	current := Reminder{ID: reminderID}
	row := tx.QueryRowContext(
		ctx,
		`SELECT name, enabled, interval_sec, break_sec, reminder_type
		 FROM reminders
		 WHERE id = ?
		   AND deleted_at IS NULL`,
		reminderID,
	)
	var enabledInt int
	switch err := row.Scan(&current.Name, &enabledInt, &current.IntervalSec, &current.BreakSec, &current.ReminderType); {
	case err == nil:
		current.Enabled = enabledInt == 1
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("reminder id %d: %w", reminderID, ErrReminderNotFound)
	default:
		return err
	}

	current, err = applyReminderPatch(current, patch)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(
		ctx,
		`UPDATE reminders
		 SET name = ?,
		     enabled = ?,
		     interval_sec = ?,
		     break_sec = ?,
		     reminder_type = ?,
		     updated_at = unixepoch()
		 WHERE id = ?
		   AND deleted_at IS NULL`,
		current.Name,
		boolToInt(current.Enabled),
		current.IntervalSec,
		current.BreakSec,
		current.ReminderType,
		current.ID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("reminder id %d: %w", reminderID, ErrReminderNotFound)
	}

	return tx.Commit()
}

func (s *HistoryStore) CreateReminder(ctx context.Context, reminder Reminder) (int64, error) {
	if err := ensureStore(ctx, s); err != nil {
		return 0, err
	}

	if err := validateReminder(reminder); err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var existingID int64
	var deletedAt sql.NullInt64
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, deleted_at
		 FROM reminders
		 WHERE name = ? COLLATE NOCASE`,
		reminder.Name,
	).Scan(&existingID, &deletedAt)
	switch {
	case err == nil && !deletedAt.Valid:
		return 0, ErrReminderAlreadyExists
	case err == nil && deletedAt.Valid:
		_, err = tx.ExecContext(
			ctx,
			`UPDATE reminders
			 SET name = ?,
			     enabled = ?,
			     interval_sec = ?,
			     break_sec = ?,
			     reminder_type = ?,
			     deleted_at = NULL,
			     updated_at = unixepoch()
			 WHERE id = ?`,
			reminder.Name,
			boolToInt(reminder.Enabled),
			reminder.IntervalSec,
			reminder.BreakSec,
			reminder.ReminderType,
			existingID,
		)
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return existingID, nil
	case errors.Is(err, sql.ErrNoRows):
	default:
		return 0, err
	}

	var res sql.Result
	if reminder.ID > 0 {
		res, err = tx.ExecContext(
			ctx,
			`INSERT INTO reminders(id, name, enabled, interval_sec, break_sec, reminder_type)
			 VALUES(?, ?, ?, ?, ?, ?)`,
			reminder.ID,
			reminder.Name,
			boolToInt(reminder.Enabled),
			reminder.IntervalSec,
			reminder.BreakSec,
			reminder.ReminderType,
		)
	} else {
		res, err = tx.ExecContext(
			ctx,
			`INSERT INTO reminders(name, enabled, interval_sec, break_sec, reminder_type)
			 VALUES(?, ?, ?, ?, ?)`,
			reminder.Name,
			boolToInt(reminder.Enabled),
			reminder.IntervalSec,
			reminder.BreakSec,
			reminder.ReminderType,
		)
	}
	if err != nil {
		return 0, err
	}
	insertedID := reminder.ID
	if insertedID <= 0 {
		insertedID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return insertedID, nil
}

func (s *HistoryStore) DeleteReminder(ctx context.Context, reminderID int64) error {
	if err := ensureStore(ctx, s); err != nil {
		return err
	}
	if reminderID <= 0 {
		return errReminderIDRequired
	}

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE reminders
		 SET deleted_at = unixepoch(),
		     updated_at = unixepoch()
		 WHERE id = ?
		   AND deleted_at IS NULL`,
		reminderID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrReminderNotFound
	}
	return nil
}

func dedupeReminderIDs(reminderIDs []int64) []int64 {
	if len(reminderIDs) == 0 {
		return nil
	}
	seen := map[int64]struct{}{}
	result := make([]int64, 0, len(reminderIDs))
	for _, raw := range reminderIDs {
		if raw <= 0 {
			continue
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		result = append(result, raw)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func (s *HistoryStore) RecordBreak(
	ctx context.Context,
	startedAt time.Time,
	endedAt time.Time,
	source string,
	plannedBreakSec int,
	actualBreakSec int,
	skipped bool,
	reminderIDs []int64,
) error {
	if err := ensureStore(ctx, s); err != nil {
		return err
	}
	if plannedBreakSec <= 0 {
		return errors.New("planned break sec must be > 0")
	}
	if actualBreakSec < 0 {
		return errors.New("actual break sec must be >= 0")
	}
	if !isValidSource(source) {
		return errors.New("invalid break source")
	}

	reasons := dedupeReminderIDs(reminderIDs)
	if len(reminderIDs) > 0 && len(reasons) != len(reminderIDs) {
		return errors.New("reminder ids must be unique positive integers")
	}

	status := statusCompleted
	skippedAtUnix := int64(0)
	if skipped {
		status = statusSkipped
		skippedAtUnix = endedAt.UTC().Unix()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(
		ctx,
		`INSERT INTO break_sessions(
		   trigger_source, status, started_at, ended_at, planned_break_sec, actual_break_sec, skipped_at
		 ) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		source,
		status,
		startedAt.UTC().Unix(),
		endedAt.UTC().Unix(),
		plannedBreakSec,
		actualBreakSec,
		nullIfZero(skippedAtUnix),
	)
	if err != nil {
		return err
	}
	sessionID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for _, reminderID := range reasons {
		name := ""
		intervalSec := plannedBreakSec
		breakSec := plannedBreakSec
		reminderType := reminderTypeRest

		row := tx.QueryRowContext(
			ctx,
			`SELECT name, interval_sec, break_sec, reminder_type
			 FROM reminders
			 WHERE id = ?
			   AND deleted_at IS NULL`,
			reminderID,
		)
		switch err := row.Scan(&name, &intervalSec, &breakSec, &reminderType); {
		case err == nil:
		case errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("reminder id %d not found", reminderID)
		default:
			return err
		}

		if err := validateReminderName(name); err != nil {
			return fmt.Errorf("invalid reminder name for id %d: %w", reminderID, err)
		}
		if intervalSec <= 0 {
			return fmt.Errorf("invalid intervalSec for reminder id %d", reminderID)
		}
		if breakSec <= 0 {
			return fmt.Errorf("invalid breakSec for reminder id %d", reminderID)
		}
		if !isValidReminderType(reminderType) {
			return fmt.Errorf("invalid reminderType for reminder id %d", reminderID)
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO break_session_reminders(
			   session_id, reminder_id, reminder_name_snapshot, interval_sec_snapshot, break_sec_snapshot, reminder_type_snapshot
			 ) VALUES(?, ?, ?, ?, ?, ?)`,
			sessionID,
			reminderID,
			name,
			intervalSec,
			breakSec,
			reminderType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func nullIfZero(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
