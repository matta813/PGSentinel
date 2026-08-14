# Health rules and scores

Every finding carries severity, confidence, evidence, likely cause, impact and a suggested investigation. Thresholds are defaults—not universal tuning truth. Current rules cover connection pressure, idle/open transactions, blockers, rollback and cache ratios, deadlocks, dead tuples, autovacuum trigger progress, large-table sequential scans, stale analyze data, high query impact, unused/duplicate index candidates, missing `pg_stat_statements`, and disabled I/O timing.

Severity deductions are deterministic: INFO 0, LOW 2, MEDIUM 6, HIGH 15, CRITICAL 35. The overall and relevant category score start at 100 and floor at zero. Thus one critical issue cannot be obscured by many informational findings. Category keys are Performance, Vacuum, Queries, Connections, Indexes, Configuration and Replication.

Query Impact Score uses logarithmically scaled components: total execution time 35%, mean latency 20%, shared block reads 20%, temporary block IO 15%, and WAL bytes 10%. This deliberately differentiates `1 ms × 20,000,000 calls` from `4 sec × 3 calls`; cumulative work can have greater capacity impact. It is a ranking signal, not a percentage or proof of causality.

Baseline comparison requires at least six observations and uses median plus three median absolute deviations. It only flags upward anomalies beyond a configured percentage. This tolerates isolated spikes better than mean/standard deviation. Future correlation must say “likely contributor” or “correlated with,” never claim causation from time series alone.

Unused indexes require zero observed scans, non-constraint status and at least 100 MiB. Scan counters reset and observation length matters; confidence is therefore Medium. Duplicate detection matches indexed columns, included columns and predicate, but operators, constraints and workload must still be checked. pgsentinel never drops an index.
