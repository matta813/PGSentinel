# Automated releases

## Source of truth

`RELEASE` contains exactly one newline-terminated Semantic Version without `v`, comments or whitespace. Stable versions use `MAJOR.MINOR.PATCH`; pre-releases such as `1.0.0-rc.1` are supported. Validation rejects prefixes, incomplete versions, leading-zero numeric components, duplicate tags, unchanged effective values, and versions whose SemVer precedence is not greater than the latest `v*` tag.

Create a release by changing the value and pushing the commit to `main`:

```bash
git switch -c release/0.4.2
printf '0.4.2\n' > RELEASE
git add RELEASE
git commit -m 'chore: release 0.4.2'
git push -u origin release/0.4.2
gh pr create --base main --fill
```

Because `main` is protected, submit the version bump through a pull request. Merging that reviewed PR produces the direct `main` event consumed by the release workflow. It is valid—and preferred—to include version-specific Compose and documentation updates in the same release PR.

## Pipeline behavior

The `CI` workflow runs Go and frontend lint, tests, and production builds for `main` and pull requests. The separate `Release` workflow runs only for a direct push to `main` where `RELEASE` changed, or an explicit recovery dispatch. It does not listen for tags, preventing a release loop. GitHub Actions concurrency serializes publication.

Publication order is verification, image build, version image push, `v` image push, optional `latest` push, then GitHub release/tag creation from the exact commit SHA. A recovery dispatch may safely repush image tags and repairs an existing release record instead of creating a duplicate. Normal release validation rejects an existing tag before expensive work.

## Release notes

The release workflow asks GitHub to generate notes for the exact range between the previous tag and the new release commit. This supplies authoritative pull request links, authors, the full comparison link, and a `New Contributors` section whenever GitHub identifies a contributor's first merged pull request.

PGSentinel then formats those entries into a Jellyfin-inspired release page with a launch heading, upgrade reminder, counted changelog, and emoji categories. Conventional pull request titles such as `feat:`, `fix(security):`, `perf:`, `docs:`, and `chore(deps):` determine the category; unmatched titles appear under General Changes. Apply the `skip-changelog` label before merging a pull request that should not appear in release notes.

## Image tags

Stable `0.4.2` produces `ghcr.io/matta813/pgsentinel:0.4.2`, `:v0.4.2`, and `:latest`. Pre-release `1.0.0-rc.1` produces only `:1.0.0-rc.1` and `:v1.0.0-rc.1`; it never changes `latest`. Pin production to a fixed version.

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
