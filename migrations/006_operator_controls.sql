CREATE TABLE maintenance_windows (
 id TEXT PRIMARY KEY, starts_at TEXT NOT NULL, ends_at TEXT NOT NULL, description TEXT NOT NULL,
 server_id TEXT REFERENCES servers(id) ON DELETE CASCADE, server_tag TEXT NOT NULL DEFAULT '',
 category TEXT NOT NULL DEFAULT '', rule_id TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 CHECK (ends_at > starts_at), CHECK (server_id IS NOT NULL OR server_tag <> '' OR category <> '' OR rule_id <> '')
);
CREATE INDEX maintenance_windows_time_idx ON maintenance_windows(starts_at, ends_at);

CREATE TABLE finding_suppressions (
 id TEXT PRIMARY KEY, finding_id TEXT REFERENCES findings(id) ON DELETE CASCADE, rule_id TEXT NOT NULL DEFAULT '',
 server_id TEXT REFERENCES servers(id) ON DELETE CASCADE, server_tag TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 CHECK (finding_id IS NOT NULL OR rule_id <> ''), CHECK (finding_id IS NOT NULL OR server_id IS NOT NULL OR server_tag <> '')
);
CREATE INDEX finding_suppressions_expiry_idx ON finding_suppressions(expires_at);

CREATE TABLE threshold_overrides (
 id TEXT PRIMARY KEY, rule_id TEXT NOT NULL, scope_type TEXT NOT NULL CHECK (scope_type IN ('global','server','tag')),
 scope_value TEXT NOT NULL DEFAULT '', value REAL NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 CHECK ((scope_type='global' AND scope_value='') OR (scope_type<>'global' AND scope_value<>'')),
 UNIQUE(rule_id, scope_type, scope_value)
);
CREATE INDEX threshold_overrides_resolution_idx ON threshold_overrides(rule_id, scope_type, scope_value);
