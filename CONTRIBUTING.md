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

## Engineering expectations

- Health findings need evidence, severity, confidence, impact, and cautious investigation guidance.
- Recommendations must not claim certainty the collected evidence cannot support.
- PGSentinel never applies destructive PostgreSQL changes automatically.
- Collectors must work with PostgreSQL 15+ or degrade explicitly by server version.
- Collection queries should be bounded and avoid adding meaningful load.
- API changes belong under `/api/v1` until a deliberate version transition.
- Go code must pass `gofmt`, `go vet`, and tests. TypeScript remains strict and free of unnecessary `any`.

## Pull requests

Pull requests require passing CI and resolved review conversations. Explain user-visible behavior, operational risk, and verification. Add documentation for new settings, collectors, rules, APIs, or deployment requirements.

The pull-request title becomes part of the next automated release notes. Use `feat:`, `fix:`, `fix(security):`, `perf:`, `docs:`, `test:`, `refactor:`, `ci:`, `build:`, or `chore(deps):` so it lands in the intended category. GitHub automatically recognizes a contributor's first merged pull request and adds the contributor to the release page. Maintainers may apply `skip-changelog` to omit internal-only work.

Maintainers may ask to split oversized changes. Dependency pull requests are reviewed manually; no dependency update is auto-merged.

By contributing, you agree that your contribution is licensed under the repository's [MIT License](LICENSE).
