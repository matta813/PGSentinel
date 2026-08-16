# PGSentinel

[![CI](https://github.com/matta813/PGSentinel/actions/workflows/ci.yml/badge.svg)](https://github.com/matta813/PGSentinel/actions/workflows/ci.yml)
[![CodeQL](https://github.com/matta813/PGSentinel/actions/workflows/codeql.yml/badge.svg)](https://github.com/matta813/PGSentinel/actions/workflows/codeql.yml)
[![GitHub release](https://img.shields.io/github/v/release/matta813/pgsentinel)](https://github.com/matta813/PGSentinel/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/matta813/PGSentinel/badge)](https://scorecard.dev/viewer/?uri=github.com/matta813/PGSentinel)

PostgreSQL monitoring and health analysis that explains **what is wrong, why it matters, the evidence behind it, and what to investigate next**. pgsentinel is deliberately an operations inbox rather than a wall of graphs.

> Early usable release. PostgreSQL 15+ is the primary target. Never apply a recommendation to production without validating it against the workload and recovery plan.

## Features

- Multiple PostgreSQL servers, encrypted credentials, SSL modes and connection diagnostics
- Connections, transactions, databases, locks, tables, vacuum, indexes and configuration collection
- `pg_stat_statements` query load with a documented multi-factor Query Impact Score
- Stable problem fingerprints with active, resolved and reopened lifecycle
- Evidence-driven rules, confidence, weighted server health and category scores
- Duplicate/unused index candidates; no automatic destructive database changes
- encrypted ntfy and generic webhook destinations with delivery tests
- low-cardinality Prometheus metrics for service and PostgreSQL health
- Responsive professional light/dark React interface
- SQLite WAL storage, migrations and 30-day raw snapshot retention
- One production container, health/readiness endpoints and GitHub Actions CI

## Quick start

Install Docker with the Compose v2 plugin, then run:

```bash
curl -fsSL https://raw.githubusercontent.com/matta813/PGSentinel/main/scripts/install-compose.sh | sh
```

The installer downloads the [ready-to-run Compose file](https://raw.githubusercontent.com/matta813/PGSentinel/main/docker-compose.quickstart.yml), generates unique encryption and administrator secrets in `pgsentinel/.env`, pulls the published image, starts it, and waits for a healthy service. It prints the generated administrator password once. Open <http://localhost:8080>, sign in, then add an existing PostgreSQL server under **Servers**. Re-running the installer preserves the existing `.env` and data volume.

To inspect the files before starting instead:

```bash
curl -fsSLO https://raw.githubusercontent.com/matta813/PGSentinel/main/docker-compose.quickstart.yml
curl -fsSLO https://raw.githubusercontent.com/matta813/PGSentinel/main/scripts/install-compose.sh
less install-compose.sh
sh install-compose.sh
```

The Compose stack starts only PGSentinel and does not provision or modify a PostgreSQL instance. Keep the generated `.env` file private and backed up: losing its encryption key makes stored PostgreSQL credentials unrecoverable.

Minimal deployment:

```yaml
services:
  pgsentinel:
    image: ghcr.io/matta813/pgsentinel:0.2.0
    container_name: pgsentinel
    restart: unless-stopped
    read_only: true
    cap_drop: [ALL]
    security_opt: [no-new-privileges:true]
    tmpfs: /tmp:rw,noexec,nosuid,nodev,size=64m
    ports: ["8080:8080"]
    volumes: ["./data:/data"]
    environment:
      TZ: Europe/Zurich
      PGSENTINEL_ENCRYPTION_KEY: "replace-with-a-long-random-secret"
      PGSENTINEL_ADMIN_PASSWORD: "replace-with-a-long-random-password"
```

The image runs as UID/GID `10001`; make bind-mounted `./data` writable by that identity.

The repository Compose file defaults to the pinned `0.2.0` image. Set `PGSENTINEL_VERSION` to choose another release. Local source builds use the explicit development override:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

See the [deployment guide](docs/deployment.md) for upgrades, persistence, backup/restore, health checks and production hardening.

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
| `PGSENTINEL_ADMIN_PASSWORD` | required | Administrator login password (minimum 12 characters) |
| `PGSENTINEL_SECURE_COOKIES` | `false` | Mark session cookies Secure; enable behind production HTTPS |
| `PGSENTINEL_NOTIFICATION_ALLOWED_HOSTS` | empty | Exact comma-separated hosts allowed for private notification targets |
| `PGSENTINEL_ALLOW_PRIVATE_NOTIFICATION_TARGETS` | `false` | Permit every private notification target; prefer the host allowlist |
| `PGSENTINEL_LISTEN_ADDR` | `:8080` | HTTP listen address |
| `PGSENTINEL_DATA_DIR` | `/data` | SQLite data directory |
| `PGSENTINEL_STATS_INTERVAL` | `30s` | Monitoring cycle interval |
| `PGSENTINEL_RETENTION` | `720h` | Configured retention horizon |
| `PGSENTINEL_LOG_LEVEL` | `info` | `info` or `debug` structured JSON logging |
| `PGSENTINEL_TRUSTED_PROXY_CIDRS` | empty | Exact reverse-proxy CIDRs allowed to supply `X-Forwarded-For` |

Passwords, tokens, and full connection URLs are never logged or returned by normal APIs. Normalized `pg_stat_statements.query` text can still contain literals for statements that PostgreSQL cannot normalize; treat database access as sensitive.

## Releases

Completed pull requests are merged individually into protected `main`; `RELEASE` stays at the latest published version while changes accumulate. GitHub later builds the release notes from every merged pull request between the previous and new tags. To publish the accumulated work, change [`RELEASE`](RELEASE) to a greater Semantic Version in a dedicated release pull request:

```bash
git switch -c release/0.3.0
printf '0.3.0\n' > RELEASE
git add RELEASE
git commit -m "chore: release 0.3.0"
git push -u origin release/0.3.0
gh pr create --base main --fill
```

After lint, tests and builds pass, GitHub Actions publishes the version, `v`-prefixed, and—for stable releases—`latest` image tags to GHCR. It creates the Git tag and a counted, categorized release page with pull-request links, authors, first-time contributors, and a full comparison link. Pre-releases such as `1.0.0-rc.1` never update `latest`. Normal feature or fix merges do not publish a container.

Pin production deployments to a version:

```bash
docker pull ghcr.io/matta813/pgsentinel:0.2.0
```

`latest` is convenient for evaluation but moves on every stable release. See [release workflow, permissions, and troubleshooting](docs/releases.md).

## Development

Requires Go 1.26, Node 24+, npm and optionally Docker.

```bash
npm ci --prefix frontend
export PGSENTINEL_ENCRYPTION_KEY=development-only-change-this-key
export PGSENTINEL_ADMIN_PASSWORD=development-only-admin-password
make test
make lint
make build
```

See the [documentation index](docs/README.md), [architecture](docs/architecture.md), [health rules](docs/health-rules.md), and [development guide](docs/development.md).

## API

Versioned endpoints live under `/api/v1`: servers and connection tests, overview, problems, historical core metrics, queries, tables, indexes, locks, vacuum, configuration, notification destination CRUD, and notification testing. Server tags are normalized and can be filtered with `GET /api/v1/servers?tag=production`. The problem inbox supports combinable `status`, `serverId`, `severity`, `category`, and `search` parameters. Metric history is available at `GET /api/v1/servers/{id}/metric-history?name=connections.total`; optional `from` and `limit` parameters constrain the series. Operational APIs require an authenticated administrator session. `GET /health`, `GET /ready`, and `GET /api/v1/version` remain public for probes and inventory.

## Roadmap and limitations

- Current collection opens a small pool per cycle and deeply inspects the connection database (`postgres`); per-database fan-out is the next collector milestone.
- Raw snapshots have simple retention; tiered downsampling and long-term aggregation are planned.
- Baseline primitives exist, while continuous per-query 24-hour regression evaluation and causal timeline correlation remain planned.
- Alert provider delivery tests work; persisted alert-routing rules and automatic dispatch remain planned.
- OS/disk metrics are intentionally absent without a reliable agent or exporter.
- EXPLAIN architecture is reserved. pgsentinel never runs `EXPLAIN ANALYZE` automatically.

The maintained roadmap and contribution priorities are in [ROADMAP.md](ROADMAP.md).

## Community

Contributions are welcome. Read the [contribution guide](CONTRIBUTING.md), use [GitHub Discussions](https://github.com/matta813/PGSentinel/discussions) for support, and report vulnerabilities according to the [security policy](SECURITY.md). Project participation follows the [Code of Conduct](CODE_OF_CONDUCT.md) and [governance model](GOVERNANCE.md).

PGSentinel is licensed under the [MIT License](LICENSE).

## Dependency updates

Dependabot monitors Go, npm, Docker/Compose and GitHub Actions dependencies through pull requests. All updates—including patches and security fixes—require manual review and merge. Product releases remain separate: Dependabot never edits `RELEASE` and cannot trigger image publication. Operational details are in the [development guide](docs/development.md#dependency-updates).
