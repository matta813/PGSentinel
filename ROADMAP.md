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
- Scoped maintenance windows, visible temporary suppressions, and safe threshold overrides with deterministic precedence.
- LSN-aware replication gaps, WAL generation and retention trends, archive failures, recovery timelines, and PostgreSQL 17+ restartpoint intelligence.

## Now

### Make data freshness explicit

- Track freshness and completeness per collector/resource instead of representing a target as only healthy or failed.
- Surface partial collections and stale snapshots in the API and UI so cached evidence cannot be mistaken for current evidence.
- Include freshness in confidence and health calculations without resolving findings merely because a collector temporarily failed.

### Expand query regression context

- Retain longer regression windows and correlate the shipped interval-delta findings with deployments and configuration history.
- Account for `pg_stat_statements` resets, minimum sample sizes, workload volume, and natural variance before creating a finding.
- Explain the compared windows and contributing impact signals rather than presenting a context-free percentage.

## Next

- Add tiered metric aggregation and configurable long-term retention.
- Correlate related findings into cautious, evidence-weighted incident timelines.
- Add a durable audit log for authentication, target configuration, routing changes, and finding state changes.
- Introduce read-only and operator roles only when multi-user workflows can retain the current simple security boundary.
- Export redacted diagnostic bundles that make support and analyzer feedback reproducible without exposing credentials or raw production queries.

## Later exploration

- Explicit, guarded `EXPLAIN (FORMAT JSON)` support for safe read-only statements.
- Optional agent-assisted host metrics with clear provenance.
- Additional notification providers and reusable rule profiles.
- Import/export of reviewed rule profiles and redacted troubleshooting data.

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
