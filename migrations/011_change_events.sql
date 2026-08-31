CREATE TABLE IF NOT EXISTS change_events (
 id TEXT PRIMARY KEY,
 server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
 kind TEXT NOT NULL,
 summary TEXT NOT NULL,
 details_json TEXT NOT NULL DEFAULT '[]',
 occurred_at TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS change_events_lookup_idx ON change_events(server_id, occurred_at DESC);
