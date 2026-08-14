# pgsentinel

PostgreSQL monitoring and health analysis that explains **what is wrong, why it matters, the evidence behind it, and what to investigate next**. pgsentinel is deliberately an operations inbox rather than a wall of graphs.

> Early usable release. PostgreSQL 15+ is the primary target. Never apply a recommendation to production without validating it against the workload and recovery plan.

## Features

- Multiple PostgreSQL servers, encrypted credentials, SSL modes and connection diagnostics
- Connections, transactions, databases, locks, tables, vacuum, indexes and configuration collection
- `pg_stat_statements` query load with a documented multi-factor Query Impact Score
- Stable problem fingerprints with active, resolved and reopened lifecycle
- Evidence-driven rules, confidence, weighted server health and category scores
- Duplicate/unused index candidates; no automatic destructive database changes
- ntfy and generic webhook delivery tests
- Responsive professional light/dark React interface
- SQLite WAL storage, migrations and 30-day raw snapshot retention
- One production container, health/readiness endpoints and GitLab CI

## Quick start

```bash
export PGSENTINEL_ENCRYPTION_KEY="$(openssl rand -base64 32)"
docker compose up --build
```

Open <http://localhost:8080>. The compose stack also starts PostgreSQL 18 with `pg_stat_statements` and a demo monitoring role. Add it in **Servers** using host `postgres-test`, port `5432`, user `pgsentinel`, password `pgsentinel-demo-only`, and SSL mode `disable`.

Minimal deployment:

```yaml
services:
  pgsentinel:
    image: registry.gitlab.scruzzi.com/root/postgresqlui:latest
    container_name: pgsentinel
    restart: unless-stopped
    ports: ["8080:8080"]
    volumes: ["./data:/data"]
    environment:
      TZ: Europe/Zurich
      PGSENTINEL_ENCRYPTION_KEY: "replace-with-a-long-random-secret"
```

The image runs as UID/GID `10001`; make bind-mounted `./data` writable by that identity. Keep the encryption key stable—losing it makes stored credentials unrecoverable.

## PostgreSQL setup

```sql
CREATE ROLE pgsentinel LOGIN PASSWORD 'use-a-strong-secret';
GRANT pg_monitor TO pgsentinel;
```

Allow the source address in `pg_hba.conf`, prefer verified TLS across networks, and see [monitoring user guidance](docs/monitoring-user.md). Query monitoring additionally needs:

```conf
shared_preload_libraries = 'pg_stat_statements'
```

After restart: `CREATE EXTENSION pg_stat_statements;`. Absence is detected and explained in the UI.

## Configuration

| Variable | Default | Meaning |
|---|---:|---|
| `PGSENTINEL_ENCRYPTION_KEY` | required | Master secret used for AES-GCM credential encryption |
| `PGSENTINEL_LISTEN_ADDR` | `:8080` | HTTP listen address |
| `PGSENTINEL_DATA_DIR` | `/data` | SQLite data directory |
| `PGSENTINEL_STATS_INTERVAL` | `30s` | Monitoring cycle interval |
| `PGSENTINEL_RETENTION` | `720h` | Configured retention horizon |
| `PGSENTINEL_LOG_LEVEL` | `info` | `info` or `debug` structured JSON logging |

Passwords, tokens, and full connection URLs are never logged or returned by normal APIs. Normalized `pg_stat_statements.query` text can still contain literals for statements that PostgreSQL cannot normalize; treat database access as sensitive.

## Development

Requires Go 1.26, Node 24+, npm and optionally Docker.

```bash
npm ci --prefix frontend
export PGSENTINEL_ENCRYPTION_KEY=development-only-change-this-key
make test
make lint
make build
```

See [architecture](docs/architecture.md), [health rules](docs/health-rules.md), and [development guide](docs/development.md).

## API

Versioned endpoints live under `/api/v1`: servers and connection tests, overview, problems, metrics, queries, tables, indexes, locks, vacuum, configuration, and notification testing. `GET /health` is a liveness probe; `GET /ready` verifies SQLite.

## Roadmap and limitations

- Current collection opens a small pool per cycle and deeply inspects the connection database (`postgres`); per-database fan-out is the next collector milestone.
- Raw snapshots have simple retention; tiered downsampling and long-term aggregation are planned.
- Baseline primitives exist, while continuous per-query 24-hour regression evaluation and causal timeline correlation remain planned.
- Alert provider delivery tests work; persisted alert-routing rules and automatic dispatch remain planned.
- OS/disk metrics are intentionally absent without a reliable agent or exporter.
- EXPLAIN architecture is reserved. pgsentinel never runs `EXPLAIN ANALYZE` automatically.

Contributions are welcome. Please run `make test lint` and keep recommendations cautious and evidence-backed. Licensed under [MIT](LICENSE).
