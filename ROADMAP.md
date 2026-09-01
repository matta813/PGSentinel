# Roadmap

PGSentinel's direction is guided by a simple test: can an administrator understand a PostgreSQL problem and choose a safe next investigation within ten seconds?

This roadmap reflects the shipped repository and recent commit history. Items are ordered by operator value and technical dependency, not promised release dates. Substantial proposals should begin in a GitHub Discussion and reach `main` only as independently tested, reviewable changes.

## Recently delivered

- A searchable operations inbox with stable finding fingerprints and active, acknowledged, resolved, and reopened lifecycle states.
- Evidence-driven health rules, weighted server/category scoring, Query Impact Score, core metric history, and Prometheus health metrics.
- Independently scheduled collectors for connections, locks, core statistics, queries, tables, indexes, vacuum, and configuration across multiple databases.
- Multi-server management with tags, SSL modes, connection diagnostics, credential rotation, and encrypted notification destinations.
- A persistent administrator account, forced bootstrap-password replacement, hardened sessions, bounded API input, and SSRF-aware notification target policy.
- A responsive light/dark interface, one-command Compose installer, hardened container defaults, protected CI, and automated release publication.
- Role-aware replication/WAL collection, conservative query-regression findings, explicit degraded collection state, and lifecycle notification delivery.
- Deterministic notification routing by finding and server attributes, with cooldowns and bounded, redacted retry history.
- Reset-aware persistent query-regression findings with explicit windows, sample sufficiency, workload deltas, and recovery handling.
- Scoped maintenance windows, visible temporary suppressions, and safe threshold overrides with deterministic precedence.
- LSN-aware replication gaps, WAL generation and retention trends, archive failures, recovery timelines, and PostgreSQL 17+ restartpoint intelligence.
- Self-contained tiered metric retention with recent raw data, 15-minute operational history, and 6-hour long-term aggregates.
- Conservative incident correlation with explained PostgreSQL relationships and chronological finding timelines.
- Durable, secret-free audit history for authentication and operator configuration changes.
- Persistent per-resource collection freshness, visible stale/partial evidence, and conservative confidence and health-score caps.
- Server-enforced administrator, operator, and viewer roles with role-change session invalidation.
- Bounded, redacted diagnostic bundles for reproducible support and analyzer feedback.
- Query-regression correlation with recorded deployments and detected PostgreSQL configuration changes.
- Reusable, validated analyzer rule profiles with previewed import, export, and scoped application.

## Now

### Wait Event Intelligence

- Collect current PostgreSQL-native wait evidence from `pg_stat_activity`, including `wait_event_type`, `wait_event`, `state`, `query_start`, `xact_start`, and `backend_type` where available.
- Expose current wait activity and conservatively aggregate the wait classes PostgreSQL reports, including Lock, IO, LWLock, Client, IPC, Timeout, and categories introduced by supported PostgreSQL versions.
- Show which wait classes dominate active pressure while retaining source provenance and collection freshness.
- Produce findings only when evidence is sufficient, and connect wait evidence with existing lock, query, and problem data only when the relationship is safe to make.
- Treat correlation as investigation context, never proof of causation.
- Add a **Wait Events** view under **Performance**, alongside Query Performance, Tables, Index Analysis, Vacuum, and Locks.

### Expand notification delivery

- Add Slack, Discord, Microsoft Teams, and SMTP/email providers through the existing notification architecture.
- Reuse routing; severity, category, server, and tag filters; cooldowns; retries; bounded retry history; redacted errors; encrypted secrets; and SSRF protections where applicable.
- Keep provider-specific transport details behind shared delivery contracts rather than duplicating routing or lifecycle behavior.

## Next

### PostgreSQL Activity Explorer

- Add a read-only `pg_stat_activity` explorer for PID, database, user, application, client address, backend type, state, query and transaction timing, state changes, wait class/event, and bounded current query text where available.
- Support focused filtering by database, user, application, backend type, state, wait type, and runtime.
- Remain evidence- and investigation-oriented: do not add controls to cancel queries, terminate backends, or kill sessions.

### Guarded Query Plan Inspector

- Support `EXPLAIN (FORMAT JSON)` only for safely validated read-only statements.
- Inspect estimates and plan structure such as sequential and index scans, estimated row counts, costs, nested loops, sorts, hashes, and plan depth.
- Keep advice cautious and evidence-based, and never run automatic `EXPLAIN ANALYZE`, because it executes the statement.

### Database Growth & Capacity Intelligence

- Retain real measurements for current and historical database size, daily and weekly growth, 30-day change, and table and index growth.
- Offer conservative linear projections toward configured capacity thresholds, described as: **“Linear estimate based on recent observed growth.”**
- Never claim that a disk will be full in a given number of days without explicit, trusted host-filesystem capacity evidence.

## Later exploration

### OIDC / SSO

- Add generic OpenID Connect authentication compatible with Authentik, Keycloak, Microsoft Entra ID, Okta, and standards-compliant providers, without necessarily removing local authentication.
- Allow explicit claim/group mapping to administrator, operator, and viewer roles.
- Require strict issuer validation, state and nonce validation, PKCE where applicable, secure callbacks, safe account linking, an explicit local-administrator recovery path, and no silent privilege escalation.

### PgBouncer Monitoring

- Add optional visibility into pools; client, server, active, idle, and waiting connections; maximum wait; utilization; pooling mode; and database/user pools.
- Consider evidence-driven findings for waiting-client growth, saturated pools, sustained pressure, and abnormal wait duration.
- Label provenance explicitly as **PgBouncer evidence** and never present it as PostgreSQL server evidence.

### Schema Change Timeline

- Observe relevant index, table, PostgreSQL configuration, and extension changes when they can be detected safely, and correlate them with deployment events already recorded by PGSentinel.
- Help operators answer “What changed before this finding started?” without promising complete DDL auditing unless explicit evidence supports it.

### Optional host metrics

- Add optional agent-assisted host metrics with explicit provenance and a clear separation from PostgreSQL-native evidence.

### Portable operational configuration

- Continue import/export work for reviewed rule profiles and bounded, redacted troubleshooting data without exporting secrets.

## Delivery requirements

Every roadmap feature must:

- preserve PGSentinel's read-only boundary toward monitored PostgreSQL servers;
- state evidence, uncertainty, operational impact, and a safe next investigation;
- handle partial collection, cancellation, restart, and version differences explicitly;
- include migrations and rollback considerations when persisted data changes;
- add regression tests, operator documentation, and bounded resource behavior;
- avoid presenting planned behavior as a shipped capability.

## Non-goals

- Automatically changing PostgreSQL configuration or schema objects.
- Automatically running `EXPLAIN ANALYZE` against captured queries.
- Fabricating host metrics that PostgreSQL cannot observe reliably.
- Replacing workload-specific testing and operator judgment with absolute tuning claims.
- Collecting product analytics or sending monitored data to a hosted service by default.

Roadmap items are intentions, not commitments. Priorities may change when production feedback, PostgreSQL version behavior, security findings, or measured implementation risk provides better evidence.
