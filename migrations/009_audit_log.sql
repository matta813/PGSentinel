PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS audit_events (
 id TEXT PRIMARY KEY,
 occurred_at TEXT NOT NULL,
 actor TEXT NOT NULL,
 action TEXT NOT NULL,
 resource_type TEXT NOT NULL,
 resource_id TEXT NOT NULL DEFAULT '',
 summary TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS audit_events_time_idx ON audit_events(occurred_at DESC, id);
CREATE INDEX IF NOT EXISTS audit_events_action_idx ON audit_events(action, occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_resource_idx ON audit_events(resource_type, resource_id, occurred_at DESC);
