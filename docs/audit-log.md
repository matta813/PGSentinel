# Audit log

PGSentinel records security-sensitive and configuration-changing operator actions in its embedded SQLite database. The Audit log page is intended for incident review and change accountability, not as an external compliance or SIEM product.

## Recorded actions

The initial action set includes successful, failed, and rate-limited logins; logout and password changes; PostgreSQL target creation, editing, removal, and credential rotation; notification destination and routing changes; maintenance windows; suppressions; scoped threshold overrides; and finding acknowledgement or reopening.

Each event contains a timestamp, actor, stable action name, resource type, resource identifier, and a fixed safe summary. Failed login events use `anonymous` and deliberately omit the submitted username and source IP.

PGSentinel never writes passwords, PostgreSQL credentials, notification URLs or tokens, encryption keys, cookies, session IDs, request bodies, or arbitrary operator-entered reasons into audit summaries. Credential rotation is recorded only as the fact that rotation occurred.

## API and search

`GET /api/v1/audit-events` supports exact `actor`, `action`, and `resourceType` filters, bounded text `search`, RFC3339 `from`/`to`, and `limit`/`offset` pagination. Limits are capped at 100 rows and offsets at 10,000. The endpoint uses the existing authenticated session boundary.

The frontend provides common action and resource filters, free-text search, safe identifier truncation, and bounded previous/next navigation.

## Retention and failure behaviour

Audit events are retained for 365 days and capped at the newest 100,000 rows. Cleanup occurs in the same SQLite transaction as each new event. This bounds growth without requiring external infrastructure.

Audit writes happen after the corresponding application mutation has succeeded. If an audit insert fails, PGSentinel emits a structured internal error containing only the action and resource identifiers; it does not log the original request or secrets, and it does not falsely report that the already completed mutation failed. Operators should alert on application errors and protect/backup the SQLite data volume.

## Migration and rollback

Migration `009_audit_log.sql` is additive and preserves existing 0.6.0 installations. Historical actions performed before this migration cannot be reconstructed and are not fabricated. Downgrading leaves the audit table in place for older versions to ignore; it is not removed automatically.
