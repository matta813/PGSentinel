# Automated releases

## Source of truth

`RELEASE` contains exactly one newline-terminated Semantic Version without `v`, comments or whitespace. Stable versions use `MAJOR.MINOR.PATCH`; pre-releases such as `1.0.0-rc.1` are supported. Validation rejects prefixes, incomplete versions, leading-zero numeric components, duplicate tags, unchanged effective values, and versions whose SemVer precedence is not greater than the latest `v*` tag.

Create a release by changing the value and pushing the commit to `main`:

```bash
git switch -c release/0.6.0
printf '0.6.0\n' > RELEASE
git add RELEASE docker-compose.yml docker-compose.quickstart.yml .env.example docs/deployment.md
git commit -m 'chore: release 0.6.0'
git push -u origin release/0.6.0
gh pr create --base main --fill
```

Because `main` is protected, submit the version bump through a pull request. Merging that reviewed PR produces the direct `main` event consumed by the release workflow. It is valid—and preferred—to include version-specific Compose and documentation updates in the same release PR.

## Accumulating changes

Normal feature, fix, documentation, security, and dependency pull requests merge independently into `main` without changing `RELEASE`. The branch therefore accumulates the next version as a sequence of small reviewed changes instead of one large release merge. Every merge must leave `main` buildable and tested; unfinished behavior should remain unmerged or be protected by a disabled-by-default feature flag.

The previous release tag is the lower boundary of the next changelog. When the release pull request finally changes `RELEASE`, GitHub compares that previous tag with the release commit and discovers every merged pull request in between. Patch maintenance for older release lines would require explicit release branches and backports; the pre-1.0 project currently supports only the latest published line.

## Pipeline behavior

The `CI` workflow runs Go and frontend lint, tests, and production builds for `main` and pull requests. The separate `Release` workflow runs only for a direct push to `main` where `RELEASE` changed, or an explicit recovery dispatch. It does not listen for tags, preventing a release loop. GitHub Actions concurrency serializes publication.

Publication order is version validation, full application verification, image build, version image push, `v` image push, optional `latest` push, then GitHub release/tag creation from the exact commit SHA. Compose semantics and the synchronization of `RELEASE`, Compose defaults, Quickstart, and `.env.example` are tested in CI. A recovery dispatch may safely repush image tags and repair an existing release record instead of creating a duplicate. Normal release validation rejects an existing tag before expensive work.

## Release notes

The release workflow asks GitHub to generate notes for the exact range between the previous tag and the new release commit. This supplies authoritative pull request links, authors, the full comparison link, and a `New Contributors` section whenever GitHub identifies a contributor's first merged pull request. No contributor file or manual list is maintained.

PGSentinel then formats those entries into a Jellyfin-inspired release page with a launch heading, upgrade reminder, counted changelog, and emoji categories for Security, Features, Bug fixes, Performance, Documentation, Dependencies, Tests, Maintenance, and General Changes. Conventional pull request titles such as `feat:`, `fix(security):`, `perf:`, `docs:`, and `chore(deps):` determine the category; unmatched titles appear under General Changes. Apply the `skip-changelog` label before merging a pull request that should not appear in release notes.

### Announcement draft

After publication, maintainers can turn the actual GitHub release body into a human-reviewable announcement draft:

```bash
gh release view v0.6.0 --json body --jq .body > /tmp/pgsentinel-0.6.0-notes.md
python3 scripts/generate-release-announcement.py \
  --version 0.6.0 \
  --title 'PGSentinel v0.6.0' \
  --notes /tmp/pgsentinel-0.6.0-notes.md \
  --release-url https://github.com/matta813/PGSentinel/releases/tag/v0.6.0 \
  --output /tmp/pgsentinel-0.6.0-announcement.md
```

The output includes install and upgrade guidance, short social copy, a project Discussion draft, and a longer community post. The [announcement workflow](../.github/workflows/release-announcement.yml) prepares the same file as a downloadable artifact after the protected release workflow succeeds. It never publishes externally: a maintainer must verify claims, choose an appropriate channel, and approve the final wording. See the [organic growth guide](growth.md) for the weekly reporting workflow and content principles.

## Image tags

Stable `0.6.0` produces `ghcr.io/matta813/pgsentinel:0.6.0`, `:v0.6.0`, and `:latest`. Pre-release `1.0.0-rc.1` produces only `:1.0.0-rc.1` and `:v1.0.0-rc.1`; it never changes `latest`. Pin production to a fixed version.

Build arguments embed version, commit SHA and UTC build time. They are visible at `GET /api/v1/version` and in the sidebar. Local builds remain `dev`, `unknown`, `unknown`.

## Authentication and protection

The release job uses GitHub's short-lived `GITHUB_TOKEN`; no long-lived release secret is required. Its job-scoped permissions are limited to `contents: write` for the tag/release and `packages: write` for GHCR. All other workflows default to `contents: read`.

Recommended settings: protect `main` with a GitHub ruleset, require the CI checks and reviews, grant the release workflow write access through `GITHUB_TOKEN`, and keep GHCR linked to this repository. Dependabot updates remain manual.

## Troubleshooting

- **Invalid version:** run `./scripts/test-release.sh` and `./scripts/validate-release.sh RELEASE`.
- **Version went backwards:** choose a version greater than the newest release under SemVer precedence.
- **Tag exists:** never reuse versions. For a partial publication, verify the tag target before repairing the release record.
- **Registry denied:** ensure GitHub Actions has `packages: write` and the package is linked to this repository.
- **Release API denied:** ensure the workflow has `contents: write` and repository Actions are allowed write access.
- **No release jobs:** ensure this is a direct push to `main` and `RELEASE` actually changed.
- **Retry after infrastructure failure:** open **Actions → Release → Run workflow**, select `main`, and enable `recover`. This uses the current workflow, safely republishes image tags, and creates or repairs the GitHub Release. Do not rerun an old job when its workflow definition itself was faulty.
