CREATE TABLE audit_log (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  action TEXT NOT NULL,
  actor TEXT NOT NULL,
  actor_name TEXT NOT NULL DEFAULT '',
  schedule_id TEXT NOT NULL DEFAULT '',
  schedule_title TEXT NOT NULL DEFAULT '',
  event_id TEXT NOT NULL DEFAULT '',
  event_name TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_audit_created ON audit_log(created_at DESC);
CREATE INDEX idx_audit_schedule ON audit_log(schedule_id, created_at DESC);
