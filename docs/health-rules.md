# Health rules and scores

Every finding carries severity, confidence, evidence, likely cause, impact and a suggested investigation. Thresholds are defaults—not universal tuning truth. Current rules cover connection pressure, idle/open transactions, blockers, rollback and cache ratios, deadlocks, dead tuples, autovacuum trigger progress, large-table sequential scans, stale analyze data, high query impact, query regressions, replication state and lag, inactive slots retaining WAL, checkpoint pressure, unused/duplicate index candidates, missing `pg_stat_statements`, and disabled I/O timing.

Severity deductions are deterministic: INFO 0, LOW 2, MEDIUM 6, HIGH 15, CRITICAL 35. The overall and relevant category score start at 100 and floor at zero. Thus one critical issue cannot be obscured by many informational findings. Category keys are Performance, Vacuum, Queries, Connections, Indexes, Configuration, Replication and WAL.

Query Impact Score uses logarithmically scaled components: total execution time 35%, mean latency 20%, shared block reads 20%, temporary block IO 15%, and WAL bytes 10%. This deliberately differentiates `1 ms × 20,000,000 calls` from `4 sec × 3 calls`; cumulative work can have greater capacity impact. It is a ranking signal, not a percentage or proof of causality.

Query regression analysis compares interval deltas rather than cumulative `pg_stat_statements` means. It requires six compatible baseline intervals with at least 20 calls each, a current interval with at least 20 calls and one second of total execution time, a baseline median of at least 5 ms, and both a 75% increase and a three-median-absolute-deviation anomaly. Counter decreases, resets, eviction, and missing query IDs exclude incompatible intervals. The rule does not run `EXPLAIN` and favors fewer high-confidence findings over weak alerts.

Replication rules are role-aware. A recovery server is checked for a running, streaming WAL receiver; a primary is checked only for standbys it can actually observe in `pg_stat_replication`. PGSentinel does not assume a primary should have a replica when PostgreSQL exposes no evidence of that intent. Replay lag becomes a Medium finding at 60 seconds. An inactive replication slot becomes High when it retains at least 1 GiB of WAL. Operators should validate delayed-replica intent and slot ownership before acting.

Checkpoint analysis waits for at least ten observed checkpoints. It reports requested-checkpoint pressure when requested checkpoints are at least 20% of the total, and frequent checkpoints when their average interval since statistics reset is below five minutes. PostgreSQL 15–16 use `pg_stat_bgwriter`; PostgreSQL 17+ use `pg_stat_checkpointer`.

If an optional collector section fails, PGSentinel preserves its last complete snapshot and does not resolve findings from incomplete evidence. The server becomes `degraded`, and every affected resource is marked `partial` or `unavailable`. A previously successful resource becomes `stale` after twice its expected interval. The UI places this state and the last success directly beside the evidence.

Finding confidence is reduced one level in API responses when its source evidence is not fresh. Estate health is capped at 75 for partial or stale evidence, 60 for an unavailable resource, and 50 for a failed or unreachable target. These caps are conservative: missing evidence must never make the displayed health look better, and a temporary collection failure must never resolve an existing finding.

`GET /api/v1/servers/{id}/freshness` returns the fixed, bounded set of collection resources. It exposes timestamps, age, expected interval, state, last success and consecutive failures. It does not expose PostgreSQL credentials or raw connection errors.

Migration 005 adds only the collection-status table and preserves all existing snapshots and findings. Downgrading to a binary that predates migration 005 leaves the table unused; removing it requires an explicit operator-managed database change and is not performed automatically.

Unused indexes require zero observed scans, non-constraint status and at least 100 MiB. Scan counters reset and observation length matters; confidence is therefore Medium. Duplicate detection matches indexed columns, included columns and predicate, but operators, constraints and workload must still be checked. pgsentinel never drops an index.
