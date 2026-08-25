CREATE TABLE collection_resource_status (
    server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('fresh', 'partial', 'unavailable')),
    last_attempt_at TEXT NOT NULL,
    last_success_at TEXT,
    expected_interval_seconds INTEGER NOT NULL CHECK (expected_interval_seconds BETWEEN 1 AND 86400),
    consecutive_failures INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    error_summary TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (server_id, resource)
);

CREATE INDEX idx_collection_resource_status_server
    ON collection_resource_status(server_id, resource);
