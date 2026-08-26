# Metric retention

PGSentinel stores monitoring history in its embedded SQLite database. It does not require Prometheus, object storage, or another service for long-term metric history.

## Retention tiers

The hourly maintenance task retains three resolutions:

| Tier | Resolution | Default retention | Purpose |
|---|---:|---:|---|
| Raw | collector interval | 24 hours | Recent incident inspection |
| Medium | 15 minutes | 30 days | Operational trends |
| Long | 6 hours | 365 days | Capacity and seasonal context |

When a raw sample becomes older than the raw window, one transaction adds it to both aggregate tiers and then removes the raw row. Each aggregate preserves the average, minimum, maximum, and source sample count. Repeating a maintenance run does not count an already removed raw sample again. Medium and long rows are deleted automatically at their configured boundaries.

`GET /api/v1/servers/{id}/metric-history` returns a chronological mix of the finest retained points. Aggregate points include `resolution`, `samples`, `minimum`, and `maximum`; raw points retain the existing response shape. Results remain bounded by the endpoint's `limit` parameter.

## Configuration

All values use Go duration syntax:

```dotenv
PGSENTINEL_METRIC_RAW_RETENTION=24h
PGSENTINEL_METRIC_MEDIUM_RETENTION=720h
PGSENTINEL_METRIC_LONG_RETENTION=8760h
```

The raw window must be between 1 hour and 30 days. The medium window must be at least as long as raw and no longer than 180 days. The long window must be at least as long as medium and no longer than 5 years. Startup rejects an unsafe ordering or out-of-range value.

`PGSENTINEL_RETENTION` remains the independent raw snapshot retention setting. Findings, targets, users, and notification history are not removed by metric retention.

## Upgrades, sizing, and rollback

Migration `007_metric_retention.sql` is additive. Existing raw metric rows remain available and are gradually tiered by the first hourly maintenance run after upgrade. For a large existing database that first transaction may do more work than later incremental runs; take the normal SQLite backup before upgrading and allow temporary free disk space for the aggregate rows.

Lower retention values bound future database growth but do not immediately reclaim the SQLite file's allocated filesystem size. SQLite reuses freed pages. PGSentinel intentionally does not run automatic `VACUUM`, which could impose a long exclusive maintenance operation.

Rolling back the application leaves the aggregate table intact. Older versions ignore it and continue to read only remaining raw points. Downgrade does not reconstruct raw samples from aggregates and does not remove the new table automatically.
