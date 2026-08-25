CREATE TABLE notification_routes (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 enabled INTEGER NOT NULL DEFAULT 1,
 priority INTEGER NOT NULL DEFAULT 100,
 severities_json TEXT NOT NULL DEFAULT '[]',
 categories_json TEXT NOT NULL DEFAULT '[]',
 server_ids_json TEXT NOT NULL DEFAULT '[]',
 server_tags_json TEXT NOT NULL DEFAULT '[]',
 transitions_json TEXT NOT NULL DEFAULT '[]',
 cooldown_seconds INTEGER NOT NULL DEFAULT 0 CHECK(cooldown_seconds BETWEEN 0 AND 86400),
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX notification_routes_name_idx ON notification_routes(name);
CREATE INDEX notification_routes_order_idx ON notification_routes(enabled,priority,id);
CREATE TABLE notification_route_destinations (
 route_id TEXT NOT NULL REFERENCES notification_routes(id) ON DELETE CASCADE,
 destination_id TEXT NOT NULL REFERENCES notification_configs(id) ON DELETE CASCADE,
 PRIMARY KEY(route_id,destination_id)
);
ALTER TABLE finding_notification_deliveries ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE finding_notification_deliveries ADD COLUMN last_attempt_at TEXT;
ALTER TABLE finding_notification_deliveries ADD COLUMN next_attempt_at TEXT;
ALTER TABLE finding_notification_deliveries ADD COLUMN skipped_reason TEXT NOT NULL DEFAULT '';
UPDATE finding_notification_deliveries SET status=CASE WHEN delivered_at IS NOT NULL THEN 'delivered' WHEN attempts >= 3 THEN 'failed' ELSE 'pending' END;
CREATE INDEX finding_notification_retry_idx ON finding_notification_deliveries(status,next_attempt_at,event_id);
