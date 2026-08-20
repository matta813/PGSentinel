# Architecture

pgsentinel ships as one container. Vite builds a static React/TypeScript application; Go serves those assets and the REST API. The backend is divided into HTTP handlers, PostgreSQL access, collectors, analyzer rules, notifications and SQLite storage.

```text
Browser -> Go REST/SPA -> SQLite
                    \-> collector scheduler -> pgx (max 2 connections/target)
                                          \-> snapshots -> analyzer -> finding lifecycle
```

SQLite runs in WAL mode with one writer connection. Embedded, ordered SQL migrations make startup deterministic. Server credentials use AES-256-GCM with a SHA-256-derived key from the externally supplied master secret. API representations omit credentials.

Operational API routes require an administrator session. The administrator password remains environment-only and is verified through Argon2id. Random session tokens are delivered in HttpOnly, SameSite=Strict cookies and stored only as hashes in process memory; restarts intentionally sign every administrator out. Login attempts are rate-limited with bounded state, and forwarded client addresses are accepted only from explicitly trusted proxy CIDRs.

Notification delivery uses a dedicated outbound policy. It rejects URL credentials and unsafe address ranges, disables ambient HTTP proxies, resolves targets at dial time, and validates redirects to reduce SSRF and DNS-rebinding risk. Exact private host allowlisting is available for trusted self-hosted providers.

Collectors are read-only and set `application_name=pgsentinel`. The scheduler currently performs a 30-second full cycle; the packages are split by cost so future scheduling can independently run fast (connections/locks), standard (databases/queries), slow (tables/indexes/vacuum), and metadata collectors. Connections use short timeouts and a pool capped at two. On slow cycles, table and index collection fans out per database: it connects to each collectible database (templates excluded, up to the configured fan-out limit, largest first), aggregates the results into a single snapshot per server, and skips databases that reject the monitoring connection.

Analyzer rules are pure transformations from snapshots to findings. A fingerprint derived from rule, server, database and resource gives an issue stable identity. Storage upserts matching issues, resolves disappeared issues and reopens recurring ones. No analyzer performs a database mutation.

## EXPLAIN safety boundary

A future endpoint can accept explicit `EXPLAIN (FORMAT JSON)` for validated read-only statements. It must parse/reject multiple or mutating statements, use a read-only transaction with timeout, and audit the request. `EXPLAIN ANALYZE` requires a separate explicit warning and is never scheduled because it executes the statement.
