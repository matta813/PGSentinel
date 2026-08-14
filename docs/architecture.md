# Architecture

pgsentinel ships as one container. Vite builds a static React/TypeScript application; Go serves those assets and the REST API. The backend is divided into HTTP handlers, PostgreSQL access, collectors, analyzer rules, notifications and SQLite storage.

```text
Browser -> Go REST/SPA -> SQLite
                    \-> collector scheduler -> pgx (max 2 connections/target)
                                          \-> snapshots -> analyzer -> finding lifecycle
```

SQLite runs in WAL mode with one writer connection. Embedded, ordered SQL migrations make startup deterministic. Server credentials use AES-256-GCM with a SHA-256-derived key from the externally supplied master secret. API representations omit credentials.

Collectors are read-only and set `application_name=pgsentinel`. The scheduler currently performs a 30-second full cycle; the packages are split by cost so future scheduling can independently run fast (connections/locks), standard (databases/queries), slow (tables/indexes/vacuum), and metadata collectors. Connections use short timeouts and a pool capped at two.

Analyzer rules are pure transformations from snapshots to findings. A fingerprint derived from rule, server, database and resource gives an issue stable identity. Storage upserts matching issues, resolves disappeared issues and reopens recurring ones. No analyzer performs a database mutation.

## EXPLAIN safety boundary

A future endpoint can accept explicit `EXPLAIN (FORMAT JSON)` for validated read-only statements. It must parse/reject multiple or mutating statements, use a read-only transaction with timeout, and audit the request. `EXPLAIN ANALYZE` requires a separate explicit warning and is never scheduled because it executes the statement.
