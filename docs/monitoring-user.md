# PostgreSQL monitoring user

The supported least-privilege setup is:

```sql
CREATE ROLE pgsentinel LOGIN PASSWORD 'replace-me';
GRANT pg_monitor TO pgsentinel;
```

`pg_monitor` exposes server statistics, settings and session activity without superuser. Configure a narrow `pg_hba.conf` source range and TLS (`verify-full` where certificates permit). Rotate the password through the server form when credential update support lands; for the current release delete/re-add the target.

For query statistics, add `pg_stat_statements` to `shared_preload_libraries`, restart PostgreSQL, then run `CREATE EXTENSION pg_stat_statements` in each monitored database. pgsentinel does not need ownership of application objects and does not run VACUUM, CREATE/DROP INDEX, terminate backends, reset statistics, or change settings.

The monitoring role may see query text and session metadata. Restrict access to pgsentinel itself and its `/data` volume, never expose it directly to an untrusted network, and use a reverse proxy with authentication and TLS. Native pgsentinel user authentication is not included in this release.
