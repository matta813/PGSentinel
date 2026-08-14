# Security policy

## Supported versions

| Version | Security updates |
|---|---|
| Latest `0.1.x` release | Supported |
| `main` | Development only |
| Older releases | Not supported |

Until PGSentinel reaches 1.0, security fixes are provided for the latest published release line only.

## Report a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/matta813/pgsentinel/security/advisories/new). Do not disclose a suspected vulnerability in a public issue, pull request, Discussion, or log excerpt.

Include the affected version, impact, reproduction steps, and a minimal proof of concept where safe. Remove PostgreSQL credentials, tokens, private queries, and production data.

You should receive an acknowledgement within five business days. The maintainer will validate the report, coordinate a fix and release, and discuss disclosure timing with the reporter. Timelines depend on severity and reproducibility; no bounty program is currently offered.

## Security model

PGSentinel stores PostgreSQL credentials encrypted with the configured master key. Operators remain responsible for protecting that key, restricting network and filesystem access, using TLS, backing up data safely, and granting only the documented `pg_monitor` role. See [deployment hardening](docs/deployment.md) and [monitoring-user permissions](docs/monitoring-user.md).
