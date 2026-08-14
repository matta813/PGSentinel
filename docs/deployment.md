# Deployment

## Requirements

PGSentinel runs as a single container and stores configuration, encrypted PostgreSQL credentials, snapshots and findings in SQLite under `/data`. It does not start or modify a PostgreSQL server. Docker Engine with the Compose plugin is sufficient.

Generate the encryption key once and store it in a password manager or secret store:

```bash
export PGSENTINEL_ENCRYPTION_KEY="$(openssl rand -base64 32)"
export PGSENTINEL_ADMIN_PASSWORD="replace-with-a-long-random-password"
```

Never rotate or lose this value without first implementing a credential re-encryption procedure. Existing saved PostgreSQL passwords cannot be decrypted with a different key.

## Start with Docker Compose

The repository Compose file pins the stable application version while allowing an explicit override:

```bash
export PGSENTINEL_VERSION=0.1.1
export PGSENTINEL_ENCRYPTION_KEY='value-from-your-secret-store'
export PGSENTINEL_ADMIN_PASSWORD='value-from-your-secret-store'
docker compose pull
docker compose up -d
docker compose ps
curl --fail http://127.0.0.1:8080/health
curl --fail http://127.0.0.1:8080/ready
```

Open `http://localhost:8080` and add an existing PostgreSQL target under **Servers**. Across Docker networks, use a DNS name or IP reachable from the PGSentinel container; `localhost` inside the container refers to PGSentinel itself.

## Version pinning and upgrades

Production deployments should use an immutable release version instead of `latest`. To upgrade:

1. Read the GitHub release notes and back up `/data`.
2. Change `PGSENTINEL_VERSION` in the deployment environment.
3. Pull and recreate the container.
4. Verify readiness and the version endpoint.

```bash
export PGSENTINEL_VERSION=0.2.0
docker compose pull
docker compose up -d
curl --fail http://127.0.0.1:8080/ready
curl --fail http://127.0.0.1:8080/api/v1/version
```

SQLite migrations run automatically when the new application starts. Do not downgrade across schema changes without restoring a compatible backup.

## Data and backups

Compose stores `/data` in the named volume `pgsentinel-data`. Stop the application before taking a file-level backup so SQLite and its WAL are consistent:

```bash
docker compose stop pgsentinel
docker compose cp pgsentinel:/data/pgsentinel.db ./pgsentinel.db.backup
docker compose start pgsentinel
```

To restore, stop the service, copy the database back, and start it again. Preserve ownership for container UID/GID `10001` when using bind mounts. Test restores regularly; an untested backup is not a recovery plan.

## Production hardening

- Keep the root filesystem read-only, drop all Linux capabilities, enable `no-new-privileges`, and mount only `/data` writable. The supplied Compose service configures these controls and a small non-executable `/tmp` tmpfs.
- Keep `PGSENTINEL_ENCRYPTION_KEY` outside Compose files and source control.
- Keep `PGSENTINEL_ADMIN_PASSWORD` outside Compose files and source control. PGSentinel stores only hashed, short-lived session tokens in memory.
- Publish port `8080` only to a trusted management network or place it behind an authenticated TLS reverse proxy.
- Restrict outbound network access to monitored PostgreSQL servers and configured notification endpoints.
- Use `verify-full` for PostgreSQL connections across untrusted networks.
- Grant the monitoring role `pg_monitor`; add privileges only for a documented feature.
- Pin the container to a release version and review release notes before upgrading.
- Back up `/data` and monitor both `/health` and `/ready`.

PGSentinel requires an administrator login and rate-limits failed attempts. Put it behind HTTPS and set `PGSENTINEL_SECURE_COOKIES=true`; network-level access control remains recommended.

Notification delivery blocks loopback, private, link-local, carrier-grade NAT, multicast and metadata destinations by default. DNS is checked at connection time and again after redirects to prevent DNS rebinding. For a self-hosted ntfy or webhook service, prefer an exact allowlist:

```bash
export PGSENTINEL_NOTIFICATION_ALLOWED_HOSTS=ntfy.internal.example
```

`PGSENTINEL_ALLOW_PRIVATE_NOTIFICATION_TARGETS=true` permits all private targets and should only be used on a tightly controlled network. URLs containing credentials are rejected; configure provider credentials in their dedicated fields.

## Local source build

The development override builds the checked-out source and marks the application as a development build:

```bash
export PGSENTINEL_ENCRYPTION_KEY=development-only-change-this-key
export PGSENTINEL_ADMIN_PASSWORD=development-only-admin-password
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

The production Compose file never builds source implicitly.

## Troubleshooting

- **Encryption key error:** set a value of at least 16 characters and keep it stable.
- **Notification target blocked:** add only the exact trusted hostname to `PGSENTINEL_NOTIFICATION_ALLOWED_HOSTS`; avoid globally allowing private targets.
- **Target reports connection refused:** verify routing from inside the container, PostgreSQL `listen_addresses`, firewall rules and `pg_hba.conf`.
- **Target works on the host but not in PGSentinel:** do not use `localhost`; use a container-reachable host name or address.
- **Readiness fails:** inspect `docker compose logs pgsentinel` and confirm the data volume is writable.
- **Image pull denied:** authenticate to GHCR if the package is private; public packages require no login.
