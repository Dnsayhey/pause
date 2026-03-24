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
	statusRunning      = "running"
	statusCompleted    = "completed"
	statusSkipped      = "skipped"
	statusCanceled     = "canceled"
	historySchemaVer   = 2
)

var (
	ErrReminderAlreadyExists = errors.New("reminder already exists")
	ErrReminderNotFound      = errors.New("reminder not found")
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

type ReminderDefinition struct {
	ID           int64
	Name         string
	Enabled      bool
	IntervalSec  int
	BreakSec     int
	ReminderType string
}

type ReminderMutation struct {
	ID           int64
	Name         *string
	Enabled      *bool
	IntervalSec  *int
	BreakSec     *int
	ReminderType *string
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

	var currentVersion int
	if err := s.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&currentVersion); err != nil {
		return fmt.Errorf("history read schema version failed: %w", err)
	}
	if currentVersion != historySchemaVer {
		if err := s.recreateSchema(ctx); err != nil {
			return fmt.Errorf("history recreate schema failed: %w", err)
		}
	} else {
		if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
			return fmt.Errorf("history migrate failed: %w", err)
		}
	}

	if err := s.cleanupDanglingRunningSessions(ctx); err != nil {
		return fmt.Errorf("history migrate cleanup running sessions failed: %w", err)
	}
	return nil
}

func (s *Store) recreateSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DROP TABLE IF EXISTS break_session_reminders`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DROP TABLE IF EXISTS break_sessions`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DROP TABLE IF EXISTS reminders`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, historySchemaVer)); err != nil {
		return err
	}
	return nil
}

func (s *Store) cleanupDanglingRunningSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE break_sessions
		 SET status = ?,
		     ended_at = COALESCE(ended_at, unixepoch()),
		     updated_at = unixepoch()
		 WHERE status = ?`,
		statusCanceled,
		statusRunning,
	)
	return err
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
		if reminder.ID > 0 {
			_, err := tx.ExecContext(
				context.Background(),
				`INSERT INTO reminders(id, name, enabled, interval_sec, break_sec, reminder_type)
				 VALUES(?, ?, ?, ?, ?, ?)
				 ON CONFLICT(id) DO UPDATE SET
				   name=excluded.name,
				   enabled=excluded.enabled,
				   interval_sec=excluded.interval_sec,
				   break_sec=excluded.break_sec,
				   reminder_type=excluded.reminder_type,
				   deleted_at=NULL,
				   updated_at=unixepoch()`,
				reminder.ID,
				reminder.Name,
				boolToInt(reminder.Enabled),
				reminder.IntervalSec,
				reminder.BreakSec,
				reminder.ReminderType,
			)
			if err != nil {
				return err
			}
			continue
		}
		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO reminders(name, enabled, interval_sec, break_sec, reminder_type)
			 VALUES(?, ?, ?, ?, ?)
			 ON CONFLICT(name) DO UPDATE SET
			   enabled=excluded.enabled,
			   interval_sec=excluded.interval_sec,
			   break_sec=excluded.break_sec,
			   reminder_type=excluded.reminder_type,
			   deleted_at=NULL,
			   updated_at=unixepoch()`,
			reminder.Name,
			boolToInt(reminder.Enabled),
			reminder.IntervalSec,
			reminder.BreakSec,
			reminder.ReminderType,
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
		`SELECT id, name, enabled, interval_sec, break_sec, reminder_type
		 FROM reminders
		 WHERE deleted_at IS NULL
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
		if mutation.ID <= 0 {
			continue
		}

		current := ReminderDefinition{ID: mutation.ID}

		row := tx.QueryRowContext(
			context.Background(),
			`SELECT name, enabled, interval_sec, break_sec, reminder_type
			 FROM reminders WHERE id = ?`,
			mutation.ID,
		)
		var enabledInt int
		switch err := row.Scan(&current.Name, &enabledInt, &current.IntervalSec, &current.BreakSec, &current.ReminderType); {
		case err == nil:
			current.Enabled = enabledInt == 1
		case errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("reminder id %d not found", mutation.ID)
		default:
			return err
		}

		if mutation.Name != nil {
			if err := validateReminderName(*mutation.Name); err != nil {
				return err
			}
			current.Name = *mutation.Name
		}
		if mutation.Enabled != nil {
			current.Enabled = *mutation.Enabled
		}
		if mutation.IntervalSec != nil {
			if *mutation.IntervalSec <= 0 {
				return errors.New("reminder intervalSec must be > 0")
			}
			current.IntervalSec = *mutation.IntervalSec
		}
		if mutation.BreakSec != nil {
			if *mutation.BreakSec <= 0 {
				return errors.New("reminder breakSec must be > 0")
			}
			current.BreakSec = *mutation.BreakSec
		}
		if mutation.ReminderType != nil {
			if !isValidReminderType(*mutation.ReminderType) {
				return errors.New("reminder reminderType must be rest or notify")
			}
			current.ReminderType = *mutation.ReminderType
		}

		_, err := tx.ExecContext(
			context.Background(),
			`UPDATE reminders
			 SET name = ?,
			     enabled = ?,
			     interval_sec = ?,
			     break_sec = ?,
			     reminder_type = ?,
			     deleted_at = NULL,
			     updated_at = unixepoch()
			 WHERE id = ?`,
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
	}

	return tx.Commit()
}

func (s *Store) CreateReminder(mutation ReminderMutation) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("history store is not initialized")
	}

	if mutation.Name == nil {
		return 0, errors.New("reminder name is required")
	}
	if err := validateReminderName(*mutation.Name); err != nil {
		return 0, err
	}
	if mutation.Enabled == nil {
		return 0, errors.New("reminder enabled is required")
	}
	if mutation.IntervalSec == nil || *mutation.IntervalSec <= 0 {
		return 0, errors.New("reminder intervalSec must be > 0")
	}
	if mutation.BreakSec == nil || *mutation.BreakSec <= 0 {
		return 0, errors.New("reminder breakSec must be > 0")
	}
	if mutation.ReminderType == nil {
		return 0, errors.New("reminder reminderType is required")
	}
	if !isValidReminderType(*mutation.ReminderType) {
		return 0, errors.New("reminder reminderType must be rest or notify")
	}

	next := ReminderDefinition{
		ID:           mutation.ID,
		Name:         *mutation.Name,
		Enabled:      *mutation.Enabled,
		IntervalSec:  *mutation.IntervalSec,
		BreakSec:     *mutation.BreakSec,
		ReminderType: *mutation.ReminderType,
	}
	if err := validateReminderName(next.Name); err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var existingID int64
	var deletedAt sql.NullInt64
	err = tx.QueryRowContext(
		context.Background(),
		`SELECT id, deleted_at
		 FROM reminders
		 WHERE name = ? COLLATE NOCASE`,
		next.Name,
	).Scan(&existingID, &deletedAt)
	switch {
	case err == nil && !deletedAt.Valid:
		return 0, ErrReminderAlreadyExists
	case err == nil && deletedAt.Valid:
		_, err = tx.ExecContext(
			context.Background(),
			`UPDATE reminders
			 SET name = ?,
			     enabled = ?,
			     interval_sec = ?,
			     break_sec = ?,
			     reminder_type = ?,
			     deleted_at = NULL,
			     updated_at = unixepoch()
			 WHERE id = ?`,
			next.Name,
			boolToInt(next.Enabled),
			next.IntervalSec,
			next.BreakSec,
			next.ReminderType,
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
	if next.ID > 0 {
		res, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO reminders(id, name, enabled, interval_sec, break_sec, reminder_type)
			 VALUES(?, ?, ?, ?, ?, ?)`,
			next.ID,
			next.Name,
			boolToInt(next.Enabled),
			next.IntervalSec,
			next.BreakSec,
			next.ReminderType,
		)
	} else {
		res, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO reminders(name, enabled, interval_sec, break_sec, reminder_type)
			 VALUES(?, ?, ?, ?, ?)`,
			next.Name,
			boolToInt(next.Enabled),
			next.IntervalSec,
			next.BreakSec,
			next.ReminderType,
		)
	}
	if err != nil {
		return 0, err
	}
	insertedID := next.ID
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

func (s *Store) DeleteReminder(reminderID int64) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	if reminderID <= 0 {
		return errors.New("reminder id is required")
	}

	res, err := s.db.ExecContext(
		context.Background(),
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

func (s *Store) StartBreak(startedAt time.Time, source string, plannedBreakSec int, reminderIDs []int64) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("history store is not initialized")
	}
	if plannedBreakSec <= 0 {
		return 0, errors.New("planned break sec must be > 0")
	}
	if !isValidSource(source) {
		return 0, errors.New("invalid break source")
	}
	plannedSec := plannedBreakSec
	reasons := dedupeReminderIDs(reminderIDs)
	if len(reminderIDs) > 0 && len(reasons) != len(reminderIDs) {
		return 0, errors.New("reminder ids must be unique positive integers")
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(
		context.Background(),
		`INSERT INTO break_sessions(trigger_source, status, started_at, planned_break_sec, actual_break_sec)
		 VALUES(?, ?, ?, ?, 0)`,
		source,
		statusRunning,
		startedAt.UTC().Unix(),
		plannedSec,
	)
	if err != nil {
		return 0, err
	}
	sessionID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, reminderID := range reasons {
		name := ""
		intervalSec := plannedSec
		breakSec := plannedSec
		reminderType := reminderTypeRest

		row := tx.QueryRowContext(
			context.Background(),
			`SELECT name, interval_sec, break_sec, reminder_type
			 FROM reminders
			 WHERE id = ?
			   AND deleted_at IS NULL`,
			reminderID,
		)
		switch err := row.Scan(&name, &intervalSec, &breakSec, &reminderType); {
		case err == nil:
		case errors.Is(err, sql.ErrNoRows):
			return 0, fmt.Errorf("reminder id %d not found", reminderID)
		default:
			return 0, err
		}

		if err := validateReminderName(name); err != nil {
			return 0, fmt.Errorf("invalid reminder name for id %d: %w", reminderID, err)
		}
		if intervalSec <= 0 {
			return 0, fmt.Errorf("invalid intervalSec for reminder id %d", reminderID)
		}
		if breakSec <= 0 {
			return 0, fmt.Errorf("invalid breakSec for reminder id %d", reminderID)
		}
		if !isValidReminderType(reminderType) {
			return 0, fmt.Errorf("invalid reminderType for reminder id %d", reminderID)
		}

		_, err = tx.ExecContext(
			context.Background(),
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
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return sessionID, nil
}

func (s *Store) CompleteBreak(sessionID int64, endedAt time.Time, actualBreakSec int) error {
	return s.finishBreak(sessionID, statusCompleted, endedAt, 0, actualBreakSec)
}

func (s *Store) SkipBreak(sessionID int64, skippedAt time.Time, actualBreakSec int) error {
	return s.finishBreak(sessionID, statusSkipped, skippedAt, skippedAt.UTC().Unix(), actualBreakSec)
}

func (s *Store) finishBreak(sessionID int64, status string, endedAt time.Time, skippedAtUnix int64, actualBreakSec int) error {
	if s == nil || s.db == nil {
		return errors.New("history store is not initialized")
	}
	if sessionID <= 0 {
		return nil
	}
	if actualBreakSec < 0 {
		return errors.New("actual break sec must be >= 0")
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
		actualBreakSec,
		nullIfZero(skippedAtUnix),
		sessionID,
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
