# Architecture

pgsentinel ships as one container. Vite builds a static React/TypeScript application; Go serves those assets and the REST API. The backend is divided into HTTP handlers, PostgreSQL access, collectors, analyzer rules, notifications and SQLite storage.

```text
Browser -> Go REST/SPA -> SQLite
                    \-> collector scheduler -> pgx (max 2 connections/target)
                                          \-> snapshots -> analyzer -> finding lifecycle
```

SQLite runs in WAL mode with one writer connection. Embedded, ordered SQL migrations make startup deterministic. Server credentials use AES-256-GCM with a SHA-256-derived key from the externally supplied master secret. API representations omit credentials.

Operational API routes require a user-bound administrator session. A new data volume bootstraps the built-in `admin` account from `PGSENTINEL_ADMIN_PASSWORD` and restricts its first session to replacing that password. Salted Argon2id credentials are persisted in SQLite; the environment value never overwrites an existing account. Random session tokens are delivered in HttpOnly, SameSite=Strict cookies and stored only as hashes in process memory; restarts intentionally sign every administrator out. Login attempts are rate-limited with bounded state, unknown usernames take the same expensive verification path, and forwarded client addresses are accepted only from explicitly trusted proxy CIDRs.

Notification delivery uses a dedicated outbound policy. It rejects URL credentials and unsafe address ranges, disables ambient HTTP proxies, resolves targets at dial time, and validates redirects to reduce SSRF and DNS-rebinding risk. Exact private host allowlisting is available for trusted self-hosted providers.

Collectors are read-only and set `application_name=pgsentinel`. The scheduler currently performs a 30-second full cycle; the packages are split by cost so future scheduling can independently run fast (connections/locks), standard (databases/queries), slow (tables/indexes/vacuum), and metadata collectors. Connections use short timeouts and a pool capped at two. On slow cycles, table and index collection fans out per database: it connects to each collectible database (templates excluded, up to the configured fan-out limit, largest first), aggregates the results into a single snapshot per server, and skips databases that reject the monitoring connection.

Analyzer rules are pure transformations from snapshots to findings. A fingerprint derived from rule, server, database and resource gives an issue stable identity. Storage upserts matching issues, resolves disappeared issues and reopens recurring ones. No analyzer performs a database mutation.

High and Critical finding lifecycle transitions are queued transactionally in SQLite for every notification destination that is enabled at that moment. New, severity-increased, reopened and resolved events are delivered independently per destination; successful deliveries are not repeated, while failed deliveries are attempted at most three times on later collection cycles. Lower-severity findings remain in the operations inbox to avoid noisy default alerting. Notifications include concise finding context and never contain stored destination credentials.

Replication collection uses `pg_is_in_recovery()` to select role-appropriate evidence. Primaries read `pg_stat_replication` and `pg_replication_slots`; replicas read `pg_stat_wal_receiver`. Checkpoint counters come from `pg_stat_bgwriter` on PostgreSQL 15–16 and `pg_stat_checkpointer` on PostgreSQL 17+. All queries use the existing read-only monitoring connection.

The latest raw evidence is available from `GET /api/v1/servers/{id}/replication` and `GET /api/v1/servers/{id}/wal` for authenticated operator tooling. The problem inbox remains the primary UI and exposes only evidence that crossed a documented rule threshold.

## EXPLAIN safety boundary

A future endpoint can accept explicit `EXPLAIN (FORMAT JSON)` for validated read-only statements. It must parse/reject multiple or mutating statements, use a read-only transaction with timeout, and audit the request. `EXPLAIN ANALYZE` requires a separate explicit warning and is never scheduled because it executes the statement.
