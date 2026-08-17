# Development

Install Go 1.25 and Node 22+, then `npm ci --prefix frontend`. Run the API with a writable data directory and a stable key:

```bash
export PGSENTINEL_DATA_DIR="$PWD/data"
export PGSENTINEL_ENCRYPTION_KEY=development-only-change-this-key
export PGSENTINEL_ADMIN_PASSWORD=development-only-admin-password
make backend
```

In another terminal, `make frontend`; Vite proxies API requests to port 8080. `make test`, `make lint`, and `make build` mirror CI. Docker integration is `make docker-up`.

The production Dockerfile uses a BuildKit cache mount for npm's download cache. This speeds up repeated image builds without copying `node_modules` or cached packages into the resulting image layers.

`PGSENTINEL_RETENTION` controls raw snapshot retention and is applied by the hourly pruning task. Use a Go duration such as `168h` or `720h`; findings and server configuration are not removed by snapshot retention.

Add migrations as ordered files under `migrations/`. Rules should be deterministic, return actionable evidence, state uncertainty, and include unit tests at threshold boundaries. Collector queries must be read-only, bounded, version-aware for PostgreSQL 15+, and avoid large unbounded result sets. Never log a server object containing its decrypted password.

API JSON requests are limited to 64 KiB, reject unknown fields, and must contain exactly one JSON value. Keep request models intentionally small; add a dedicated streaming endpoint if a future feature needs larger payloads.

## Collector schedules

Collector work is split across four independently configurable schedules. `PGSENTINEL_FAST_INTERVAL` refreshes connections and locks, `PGSENTINEL_STATS_INTERVAL` refreshes core statistics and query data, `PGSENTINEL_SLOW_INTERVAL` refreshes tables and indexes, and `PGSENTINEL_META_INTERVAL` refreshes server configuration. Startup performs one complete cycle so every resource is immediately populated; partial cycles reuse the latest snapshots when running the analyzer.

Use Conventional Commits for commits and pull-request titles. Release notes use the pull-request title to select their category, retain the PR author and link, and let GitHub identify first-time contributors. Apply `skip-changelog` before merge only when a pull request should be absent from public release notes. Keep generated output, databases, `.env` and credentials out of Git. Validate with `git diff --check` before pushing.

Merge completed, independently releasable pull requests into protected `main`; do not change `RELEASE` in normal feature, fix, documentation, or dependency work. `main` accumulates the next release while remaining buildable. Large incomplete work should stay in a pull request or behind a safe, disabled-by-default feature flag. A dedicated release pull request updates `RELEASE`, Compose pins, and version-specific documentation when the accumulated changes are ready to publish. See the [release workflow](releases.md).

Credential encryption has a native Go fuzz target covering arbitrary plaintext round trips and ciphertext tampering. Run a bounded campaign locally with:

```bash
go test ./internal/storage -run=^$ -fuzz=^FuzzCipherRoundTrip$ -fuzztime=30s
```

GitHub Actions runs the same campaign for relevant pull requests, weekly, and on manual dispatch.

## Dependency updates

Dependencies are monitored by GitHub's native Dependabot using [`.github/dependabot.yml`](../.github/dependabot.yml). It checks Go modules and checksums, npm manifests and `package-lock.json`, Dockerfile/Compose images, and GitHub Actions every day at 04:00 Europe/Zurich. All updates use `dependabot/*` pull requests; the bot never commits directly to `main`.

- Patch updates are grouped by ecosystem, but always wait for human review.
- Minor and major updates also always wait for human review.
- GitHub creates security update pull requests independently of the version-update schedule when Dependabot security updates are enabled.
- Manifest and lockfile changes are committed together in the same pull request.

At most five pull requests per ecosystem remain open. Dependabot uses the package manager's existing version constraints and does not convert the project wholesale to exact pins.

GitHub Actions and Docker base images are pinned to immutable commit SHAs or image digests. The readable version comment or image tag remains alongside the pin, and Dependabot updates both together. This prevents a moved upstream tag from changing a trusted CI or release build without review.

The root `osv-scanner.toml` contains the documented exception for `GO-2026-5932`. That advisory applies to the unmaintained `golang.org/x/crypto/openpgp` package, while PGSentinel uses only `golang.org/x/crypto/argon2`. Workflow tests reject any future OpenPGP import so the exception cannot silently outlive its justification. Review and remove security exceptions whenever dependency usage changes.

`RELEASE` is not a dependency manifest and is absent from every Dependabot ecosystem. Dependabot never changes product versions, creates release tags, or publishes release images. Pull requests run lint, tests, Compose validation, and application builds, while the release workflow requires a direct `main` push changing `RELEASE`.

Dependabot never enables auto-merge and no merge token or Actions secret is required. Protect `main` with a GitHub ruleset, require the CI jobs as status checks, and review every dependency pull request before merging it.
