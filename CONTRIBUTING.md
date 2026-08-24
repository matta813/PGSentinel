# Contributing to PGSentinel

Thank you for improving PGSentinel. Contributions should help an administrator understand a PostgreSQL problem, its evidence, its likely impact, and the safest next investigation step.

## Before you start

- Use a GitHub Discussion for support or an early design question.
- Search existing issues before opening a bug or proposal.
- Open an issue before a large architectural change so effort is not duplicated.
- Report vulnerabilities privately according to [SECURITY.md](SECURITY.md).

## Development workflow

1. Fork the repository and create a focused branch from `main`.
2. Follow the setup in [docs/development.md](docs/development.md).
3. Add or update tests for behavioral changes.
4. Run `make test lint build` before opening a pull request.
5. Use a Conventional Commit-style pull request title, such as `feat: add replication slot analysis`.

Keep commits reviewable and avoid unrelated formatting changes. Never commit credentials, database dumps, production queries, or tokens.

## Branch naming

Use descriptive branch names with prefixes:
- `feat/` — new features
- `fix/` — bug fixes
- `fix(security)/` — security fixes
- `perf/` — performance improvements
- `docs/` — documentation only
- `test/` — test additions
- `refactor/` — code restructuring
- `ci/` — CI/CD changes
- `build/` — build system changes
- `chore/` — maintenance tasks

Example: `feat/add-replication-slot-analysis`

## Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types: `feat`, `fix`, `fix(security)`, `perf`, `docs`, `test`, `refactor`, `ci`, `build`, `chore`

Examples:
```
feat(analyzer): add dead tuple ratio threshold configuration
fix(storage): handle SQLite busy timeout on concurrent writes
perf(collector): parallelize server collection with worker pool
docs: update deployment guide for Kubernetes
```

## Code standards

### Go
- `gofmt` formatting (enforced by CI)
- `go vet` passes
- `golangci-lint` passes (see `.golangci.yml`)
- Tests pass with `-race` flag
- No unnecessary global state
- Structured errors using `internal/errors` package
- Context propagation for cancellation

### TypeScript/React
- Strict TypeScript (`noImplicitAny`, `strictNullChecks`)
- ESLint passes with React hooks rules enabled
- No unnecessary `any` types
- Component props typed with interfaces
- Tests with Vitest and React Testing Library

## Testing expectations

- Unit tests for new business logic (analyzers, collectors, storage)
- Integration tests for API endpoints
- Frontend component tests for new UI components
- Test edge cases: empty results, errors, timeouts, context cancellation
- Run `make test` locally before pushing

## Pull requests

Pull requests require passing CI and resolved review conversations. Explain:
- User-visible behavior changes
- Operational risk and rollback plan
- How you verified the change

Add documentation for:
- New settings
- New collectors or metrics
- New health rules
- New API endpoints
- Deployment requirements

The pull-request title becomes part of the next automated release notes. Use the commit type prefixes so it lands in the intended category. GitHub automatically recognizes a contributor's first merged pull request and adds the contributor to the release page. Maintainers may apply `skip-changelog` to omit internal-only work.

Maintainers may ask to split oversized changes. Dependency pull requests are reviewed manually; no dependency update is auto-merged.

## Review guidelines

When reviewing:
- Check for security implications (input validation, auth, encryption)
- Verify tests cover the change
- Ensure no breaking API changes without version bump
- Confirm documentation is updated
- Look for performance regressions

## Release process

1. Create `release/x.y.z` branch from `main`
2. Update `RELEASE`, Compose image pins, `.env.example`, and version-specific deployment examples
3. Open PR targeting `main`
4. After CI passes and approval, merge
5. GitHub Actions builds, publishes, and creates release

By contributing, you agree that your contribution is licensed under the repository's [MIT License](LICENSE).
