-- Bind parameters:
--   :week_start = inclusive UTC unix epoch seconds
--   :week_end   = exclusive UTC unix epoch seconds
--
-- Example (SQLite shell):
--   .param set :week_start 1741536000
--   .param set :week_end   1742140800

WITH sessions_in_week AS (
  SELECT
    id,
    status,
    actual_break_sec
  FROM break_sessions
  WHERE started_at >= :week_start
    AND started_at < :week_end
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
ORDER BY triggered_count DESC, r.name COLLATE NOCASE ASC;

-- Optional app-level weekly summary.
WITH sessions_in_week AS (
  SELECT
    id,
    status,
    actual_break_sec
  FROM break_sessions
  WHERE started_at >= :week_start
    AND started_at < :week_end
    AND status <> 'running'
)
SELECT
  COUNT(id) AS total_sessions,
  COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS total_completed,
  COALESCE(SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END), 0) AS total_skipped,
  COALESCE(SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END), 0) AS total_canceled,
  COALESCE(SUM(CASE WHEN status = 'completed' THEN actual_break_sec ELSE 0 END), 0) AS total_actual_break_sec,
  COALESCE(ROUND(AVG(CASE WHEN status = 'completed' THEN actual_break_sec END), 1), 0) AS avg_actual_break_sec
FROM sessions_in_week;
