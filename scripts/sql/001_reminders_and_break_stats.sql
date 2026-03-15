PRAGMA foreign_keys = ON;

-- User-configurable reminders.
CREATE TABLE IF NOT EXISTS reminders (
  id             TEXT PRIMARY KEY,
  name           TEXT NOT NULL CHECK (length(trim(name)) > 0),
  enabled        INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  interval_sec   INTEGER NOT NULL CHECK (interval_sec > 0),
  break_sec      INTEGER NOT NULL CHECK (break_sec > 0),
  delivery_type  TEXT NOT NULL DEFAULT 'overlay'
                 CHECK (delivery_type IN ('overlay', 'notification')),
  created_at     INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at     INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_reminders_enabled ON reminders(enabled);

-- One break session can be triggered by one or more reminders.
CREATE TABLE IF NOT EXISTS break_sessions (
  id                TEXT PRIMARY KEY,
  trigger_source    TEXT NOT NULL
                    CHECK (trigger_source IN ('scheduled', 'manual')),
  status            TEXT NOT NULL
                    CHECK (status IN ('running', 'completed', 'skipped', 'canceled')),
  started_at        INTEGER NOT NULL,
  ended_at          INTEGER,
  planned_break_sec INTEGER NOT NULL CHECK (planned_break_sec > 0),
  actual_break_sec  INTEGER NOT NULL DEFAULT 0 CHECK (actual_break_sec >= 0),
  skipped_at        INTEGER,
  created_at        INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at        INTEGER NOT NULL DEFAULT (unixepoch()),
  CHECK (
    (status = 'running' AND ended_at IS NULL) OR
    (status <> 'running' AND ended_at IS NOT NULL)
  )
);

CREATE INDEX IF NOT EXISTS idx_break_sessions_started_at
  ON break_sessions(started_at);

CREATE INDEX IF NOT EXISTS idx_break_sessions_status_started_at
  ON break_sessions(status, started_at);

-- Snapshot fields keep historical reports stable even if reminder settings change later.
CREATE TABLE IF NOT EXISTS break_session_reminders (
  session_id              TEXT NOT NULL,
  reminder_id             TEXT NOT NULL,
  reminder_name_snapshot  TEXT NOT NULL,
  interval_sec_snapshot   INTEGER NOT NULL CHECK (interval_sec_snapshot > 0),
  break_sec_snapshot      INTEGER NOT NULL CHECK (break_sec_snapshot > 0),
  delivery_type_snapshot  TEXT NOT NULL
                          CHECK (delivery_type_snapshot IN ('overlay', 'notification')),
  PRIMARY KEY (session_id, reminder_id),
  FOREIGN KEY (session_id) REFERENCES break_sessions(id) ON DELETE CASCADE,
  FOREIGN KEY (reminder_id) REFERENCES reminders(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_break_session_reminders_reminder_id
  ON break_session_reminders(reminder_id);

CREATE INDEX IF NOT EXISTS idx_break_session_reminders_session_id
  ON break_session_reminders(session_id);

-- Default reminder types used by the app today.
INSERT OR IGNORE INTO reminders (
  id,
  name,
  enabled,
  interval_sec,
  break_sec,
  delivery_type
) VALUES
  ('eye', '护眼', 1, 1200, 20, 'overlay'),
  ('stand', '站立', 1, 3600, 300, 'overlay');

-- Keep updated_at fresh.
CREATE TRIGGER IF NOT EXISTS trg_reminders_updated_at
AFTER UPDATE ON reminders
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE reminders
  SET updated_at = unixepoch()
  WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_break_sessions_updated_at
AFTER UPDATE ON break_sessions
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE break_sessions
  SET updated_at = unixepoch()
  WHERE id = NEW.id;
END;
