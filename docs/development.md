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
