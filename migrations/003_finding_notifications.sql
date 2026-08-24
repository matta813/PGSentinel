CREATE TABLE finding_notification_events (
 id TEXT PRIMARY KEY,
 finding_id TEXT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
 event_type TEXT NOT NULL,
 finding_json TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE finding_notification_deliveries (
 event_id TEXT NOT NULL REFERENCES finding_notification_events(id) ON DELETE CASCADE,
 destination_id TEXT NOT NULL REFERENCES notification_configs(id) ON DELETE CASCADE,
 attempts INTEGER NOT NULL DEFAULT 0,
 delivered_at TEXT,
 last_error TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(event_id,destination_id)
);
CREATE INDEX finding_notification_pending_idx ON finding_notification_deliveries(delivered_at,event_id);
