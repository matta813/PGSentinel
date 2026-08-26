PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS incidents (
 id TEXT PRIMARY KEY,
 server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
 title TEXT NOT NULL,
 summary TEXT NOT NULL,
 rationale_json TEXT NOT NULL DEFAULT '[]',
 severity TEXT NOT NULL,
 status TEXT NOT NULL CHECK (status IN ('active','resolved')),
 started_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS incidents_list_idx ON incidents(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS incidents_server_idx ON incidents(server_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS incident_findings (
 incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
 finding_id TEXT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
 relationship TEXT NOT NULL,
 PRIMARY KEY (incident_id, finding_id)
);

CREATE INDEX IF NOT EXISTS incident_findings_finding_idx ON incident_findings(finding_id, incident_id);
