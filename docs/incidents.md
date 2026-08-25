# Incident correlation

PGSentinel groups a small set of findings when the overlap can help an operator reconstruct an event. An incident is an evidence index, not a root-cause claim.

## Correlation boundary

Two findings must belong to the same PostgreSQL server, begin no more than 15 minutes apart, and share the same relevant resource or match an explicit PostgreSQL operational relationship. Temporal proximity alone never creates an incident.

The initial relationship set covers connection pressure with locks or transactions, blocking with queries or transactions, replication with WAL, long transactions with vacuum, and query workload with performance pressure. Multiple signals from the same connection, lock, replication, transaction, vacuum, or WAL subsystem can also overlap.

Connected evidence is grouped with a human-readable explanation of every relationship used. PGSentinel deliberately says that events “occurred during the same period” and “may be related.” It does not state that one finding caused another.

## Timeline and lifecycle

The incident detail page shows each finding's first-observed and resolved timestamps in chronological order. Links return to the complete finding evidence and safe investigation steps in Problems. An incident remains active while at least one grouped finding is active or acknowledged, and resolves after its correlated findings resolve or the conservative correlation no longer applies.

Correlation runs only after a complete analyzer cycle. A degraded or partial collection does not rebuild or silently resolve incidents. Candidate input is capped at the latest 500 open or recently resolved findings per server. List APIs are paginated and capped at 100 incidents per request.

Resolved incidents are retained for 90 days. The timeline contains finding lifecycle evidence currently held by PGSentinel; it does not fabricate deployment, host, or configuration-change events that the application did not observe.

## API

- `GET /api/v1/incidents?status=active&serverId=<uuid>&limit=50&offset=0` lists bounded summaries.
- `GET /api/v1/incidents/{id}` returns grouping rationale, findings, and the chronological timeline.

Both endpoints use the existing authenticated `/api/v1` session boundary. Responses contain no target credentials or raw notification secrets.

## Migration and rollback

Migration `008_incidents.sql` adds incident summaries and their finding relationships without modifying existing findings. Removing or downgrading the application leaves those additive tables in place; older versions ignore them. Incident records can be rebuilt from retained finding lifecycle data, but a downgrade does not remove the tables automatically.
