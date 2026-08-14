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

Open <http://localhost:8080>, then add an existing PostgreSQL server under **Servers**. The Compose stack starts only PGSentinel and does not provision or modify a PostgreSQL instance.

Minimal deployment:

```yaml
services:
  pgsentinel:
    image: registry.gitlab.scruzzi.com/root/postgresqlui:0.1.0
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

## Releases

[`RELEASE`](RELEASE) is the single source of truth. To publish, change its sole line to a greater Semantic Version and push that commit to `main`:

```bash
printf '0.3.0\n' > RELEASE
git add RELEASE
git commit -m "chore: release 0.3.0"
git push origin main
```

After lint, tests and builds pass, GitLab exclusively for that version change publishes `:0.3.0`, `:v0.3.0`, and—for stable versions—`:latest`. It then creates Git tag and GitLab Release `v0.3.0` with grouped Conventional Commit notes. Pre-releases such as `1.0.0-rc.1` never update `latest`. Normal commits, merge requests, branches and tag pipelines do not build a container.

Pin production deployments to a version:

```bash
docker pull registry.gitlab.scruzzi.com/root/postgresqlui:0.3.0
```

`latest` is convenient for evaluation but moves on every stable release. See [release workflow, permissions, and troubleshooting](docs/releases.md).

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

## Dependency updates

Renovate monitors Go, npm, Docker/Compose and GitLab CI dependencies through merge requests. Patch updates can merge only after the complete protected GitLab CI succeeds; minor updates require review and major updates require Dependency Dashboard approval. Product releases remain separate: Renovate explicitly ignores `RELEASE` and cannot trigger release image publication. Operational details are in the [development guide](docs/development.md#dependency-updates).
