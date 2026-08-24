# Security policy

## Supported versions

| Version | Security updates |
|---|---|
| Latest `0.5.x` release | Supported |
| `main` | Development only |
| Older releases | Not supported |

Until PGSentinel reaches 1.0, security fixes are provided for the latest published release line only.

## Report a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/matta813/PGSentinel/security/advisories/new). Do not disclose a suspected vulnerability in a public issue, pull request, Discussion, or log excerpt.

Include the affected version, impact, reproduction steps, and a minimal proof of concept where safe. Remove PostgreSQL credentials, tokens, private queries, and production data.

You should receive an acknowledgement within five business days. The maintainer will validate the report, coordinate a fix and release, and discuss disclosure timing with the reporter. Timelines depend on severity and reproducibility; no bounty program is currently offered.

## Security model

PGSentinel stores PostgreSQL credentials encrypted with the configured master key. Administrator passwords are verified with Argon2id; random session tokens are stored only as hashes in memory and are invalidated by a restart. Notification targets are checked at connection time against private, loopback, link-local, metadata, multicast, and carrier-grade NAT ranges unless an operator explicitly allows them.

Operators remain responsible for protecting the master key and administrator password, restricting network and filesystem access, using TLS and `Secure` cookies, backing up data safely, and granting only the documented `pg_monitor` role. See [deployment hardening](docs/deployment.md) and [monitoring-user permissions](docs/monitoring-user.md).
