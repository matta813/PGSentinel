# PostgreSQL monitoring user

The supported least-privilege setup is:

```sql
CREATE ROLE pgsentinel LOGIN PASSWORD 'replace-me';
GRANT pg_monitor TO pgsentinel;
```

`pg_monitor` exposes server statistics, settings and session activity without superuser. This includes the `pg_stat_activity` fields used by Wait Events: database, user, application, client address, backend type, state, query and transaction timestamps, and PostgreSQL's current wait class and event. Configure a narrow `pg_hba.conf` source range and TLS (`verify-full` where certificates permit). Use **Edit** on the Servers page to rotate a password without deleting the monitoring target; leave the password field empty to retain the current credential.

For query statistics, add `pg_stat_statements` to `shared_preload_libraries`, restart PostgreSQL, then run `CREATE EXTENSION pg_stat_statements` in each monitored database. PostgreSQL 15+ exposes `pg_stat_statements_info.stats_reset`, which PGSentinel combines with `pg_postmaster_start_time()` to prevent baselines from crossing a statistics reset or restart. PGSentinel does not need ownership of application objects and does not run VACUUM, CREATE/DROP INDEX, terminate backends, reset statistics, change settings, or execute `EXPLAIN ANALYZE`.

The monitoring role may see query text and session metadata. Restrict access to PGSentinel and its `/data` volume and never expose it directly to an untrusted network. PGSentinel requires its native administrator login; production deployments should additionally use TLS, network-level access control, and `Secure` session cookies. Configure trusted proxy CIDRs explicitly when a reverse proxy supplies client addresses.

Wait Events is a current, cluster-wide snapshot collected once per fast cycle from the server connection; it does not connect to every database. Query text is trimmed and bounded to 2,000 bytes before persistence. PostgreSQL exposes `query_start`, `xact_start`, and `state_change`, but not a portable timestamp for when the current `wait_event` began. PGSentinel therefore labels derived durations as query age, transaction age, and state age—not wait duration.
