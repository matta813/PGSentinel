# Roadmap

PGSentinel's direction is guided by a simple test: can an administrator understand a PostgreSQL problem and choose a safe next investigation within ten seconds?

## Current priorities

- Expand per-database collection without creating excessive connection or query load.
- Add tiered metric aggregation and configurable long-term retention.
- Evaluate query regressions continuously against historical baselines.
- Persist alert routing rules and dispatch active finding transitions.
- Improve replication, WAL, and checkpoint analysis across PostgreSQL 15+.
- Correlate related findings into cautious, evidence-weighted incident timelines.

## Later exploration

- Explicit, guarded `EXPLAIN (FORMAT JSON)` support for safe read-only statements.
- Optional agent-assisted host metrics with clear provenance.
- Additional notification providers and reusable rule profiles.
- Import/export of redacted diagnostic bundles.

## Non-goals

- Automatically changing PostgreSQL configuration or schema objects.
- Automatically running `EXPLAIN ANALYZE` against captured queries.
- Fabricating host metrics that PostgreSQL cannot observe reliably.
- Replacing workload-specific testing and operator judgment with absolute tuning claims.

Roadmap items are intentions, not commitments or implemented features. Use a GitHub Discussion before investing in a substantial proposal.
