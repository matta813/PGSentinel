PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS servers (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, host TEXT NOT NULL, port INTEGER NOT NULL DEFAULT 5432, username TEXT NOT NULL,
 password_cipher BLOB NOT NULL, ssl_mode TEXT NOT NULL DEFAULT 'prefer', version TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'unknown',
 last_connected_at TEXT, last_error TEXT NOT NULL DEFAULT '', tags_json TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS servers_name_idx ON servers(name);
CREATE TABLE IF NOT EXISTS metrics (
 id INTEGER PRIMARY KEY AUTOINCREMENT, server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE, database_name TEXT NOT NULL DEFAULT '',
 name TEXT NOT NULL, value REAL NOT NULL, labels_json TEXT NOT NULL DEFAULT '{}', collected_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS metrics_lookup_idx ON metrics(server_id, name, collected_at DESC);
CREATE TABLE IF NOT EXISTS snapshots (
 id INTEGER PRIMARY KEY AUTOINCREMENT, server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE, kind TEXT NOT NULL,
 payload_json TEXT NOT NULL, collected_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS snapshots_lookup_idx ON snapshots(server_id, kind, collected_at DESC);
CREATE TABLE IF NOT EXISTS findings (
 id TEXT PRIMARY KEY, rule_id TEXT NOT NULL, fingerprint TEXT NOT NULL UNIQUE, server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
 database_name TEXT NOT NULL DEFAULT '', resource TEXT NOT NULL DEFAULT '', severity TEXT NOT NULL, category TEXT NOT NULL, title TEXT NOT NULL,
 summary TEXT NOT NULL, cause TEXT NOT NULL, impact TEXT NOT NULL, evidence_json TEXT NOT NULL, suggestions_json TEXT NOT NULL,
 confidence TEXT NOT NULL, status TEXT NOT NULL, started_at TEXT NOT NULL, updated_at TEXT NOT NULL, resolved_at TEXT
);
CREATE INDEX IF NOT EXISTS findings_inbox_idx ON findings(status, severity, updated_at DESC);
CREATE TABLE IF NOT EXISTS notification_configs (id TEXT PRIMARY KEY, provider TEXT NOT NULL, name TEXT NOT NULL, config_cipher BLOB NOT NULL, enabled INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value_json TEXT NOT NULL, updated_at TEXT NOT NULL);
