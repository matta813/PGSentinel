# Development

Install Go 1.26 and Node 24+, then `npm ci --prefix frontend`. Run the API with a writable data directory and a stable key:

```bash
export PGSENTINEL_DATA_DIR="$PWD/data"
export PGSENTINEL_ENCRYPTION_KEY=development-only-change-this-key
make backend
```

In another terminal, `make frontend`; Vite proxies API requests to port 8080. `make test`, `make lint`, and `make build` mirror CI. Docker integration is `make docker-up`.

Add migrations as ordered files under `migrations/`. Rules should be deterministic, return actionable evidence, state uncertainty, and include unit tests at threshold boundaries. Collector queries must be read-only, bounded, version-aware for PostgreSQL 15+, and avoid large unbounded result sets. Never log a server object containing its decrypted password.

Use Conventional Commits. Keep generated output, databases, `.env` and credentials out of Git. Validate with `git diff --check` before pushing.

## Dependency updates

Dependencies are monitored by a scheduled self-hosted Renovate job in this project's GitLab pipeline. The separate `root/renovate-runner` repository remains a reusable central-runner template, but the active schedule is colocated because the least-privilege Project Access Token is intentionally scoped to PGSentinel. Renovate reads [`renovate.json`](../renovate.json), updates Go modules and checksums, npm manifests and `package-lock.json`, Dockerfile/Compose images, and GitLab CI images through `renovate/*` merge requests.

- Patch, digest, and pin updates may use GitLab platform automerge only after every required pipeline and branch-protection condition succeeds.
- Minor updates always wait for human review.
- Major updates require explicit approval in the Dependency Dashboard before Renovate creates an MR.
- Vulnerability remediation bypasses normal schedules and rate limits, carries the `security` label, but major security updates are not blindly automerged.
- Lockfile maintenance runs at most monthly and requires review.

Normal updates wait three days after upstream publication. The existing `rangeStrategy: auto` preserves the project's range style instead of converting every dependency to an exact pin. Five Renovate MRs may be open concurrently and no more than two are created per hour.

`RELEASE` is explicitly ignored and has no custom/regex manager. Renovate never changes product versions, creates release tags, or publishes release images. Because release CI jobs require a direct `main` push that changes only the effective `RELEASE` version, Renovate MRs run lint, tests, and application build without a Docker release.

Automerge never bypasses approvals, merge conflicts, protected branches, or required GitLab pipelines. If project settings require approvals that the bot cannot satisfy, patch MRs remain open; retain those protections and merge manually.
