PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS metric_aggregates (
 tier TEXT NOT NULL CHECK (tier IN ('medium','long')),
 server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
 database_name TEXT NOT NULL DEFAULT '',
 name TEXT NOT NULL,
 labels_json TEXT NOT NULL DEFAULT '{}',
 bucket_start TEXT NOT NULL,
 minimum REAL NOT NULL,
 maximum REAL NOT NULL,
 value_sum REAL NOT NULL,
 sample_count INTEGER NOT NULL CHECK (sample_count > 0),
 PRIMARY KEY (tier, server_id, database_name, name, labels_json, bucket_start)
);

CREATE INDEX IF NOT EXISTS metric_aggregates_lookup_idx
 ON metric_aggregates(server_id, name, bucket_start DESC, tier);

CREATE INDEX IF NOT EXISTS metrics_retention_idx ON metrics(collected_at);
CREATE INDEX IF NOT EXISTS metric_aggregates_retention_idx ON metric_aggregates(tier, bucket_start);
