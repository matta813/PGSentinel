# PostgreSQL monitoring user

The supported least-privilege setup is:

```sql
CREATE ROLE pgsentinel LOGIN PASSWORD 'replace-me';
GRANT pg_monitor TO pgsentinel;
```

`pg_monitor` exposes server statistics, settings and session activity without superuser. Configure a narrow `pg_hba.conf` source range and TLS (`verify-full` where certificates permit). Use **Edit** on the Servers page to rotate a password without deleting the monitoring target; leave the password field empty to retain the current credential.

For query statistics, add `pg_stat_statements` to `shared_preload_libraries`, restart PostgreSQL, then run `CREATE EXTENSION pg_stat_statements` in each monitored database. pgsentinel does not need ownership of application objects and does not run VACUUM, CREATE/DROP INDEX, terminate backends, reset statistics, or change settings.

The monitoring role may see query text and session metadata. Restrict access to PGSentinel and its `/data` volume and never expose it directly to an untrusted network. PGSentinel requires its native administrator login; production deployments should additionally use TLS, network-level access control, and `Secure` session cookies. Configure trusted proxy CIDRs explicitly when a reverse proxy supplies client addresses.
