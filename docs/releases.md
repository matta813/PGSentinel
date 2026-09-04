# Automated releases

## Source of truth

`RELEASE` contains exactly one newline-terminated Semantic Version without `v`, comments or whitespace. Stable versions use `MAJOR.MINOR.PATCH`; pre-releases such as `1.0.0-rc.1` are supported. Validation rejects prefixes, incomplete versions, leading-zero numeric components, mismatched existing tags, unchanged effective values, and versions whose SemVer precedence is not greater than the latest valid PGSentinel release tag. Tag selection uses Semantic Version precedence rather than Git's `version:refname` ordering, so a stable version correctly follows its release candidates.

Create a release by changing the value and pushing the commit to `main`:

```bash
git switch -c release/0.7.0
printf '0.7.0\n' > RELEASE
git add RELEASE docker-compose.yml docker-compose.quickstart.yml .env.example docs/deployment.md
git commit -m 'chore: release 0.7.0'
git push -u origin release/0.7.0
gh pr create --base main --fill
```

Because `main` is protected, submit the version bump through a pull request. Merging that reviewed PR produces the direct `main` event consumed by the release workflow. It is valid—and preferred—to include version-specific Compose and documentation updates in the same release PR.

## Accumulating changes

Normal feature, fix, documentation, security, and dependency pull requests merge independently into `main` without changing `RELEASE`. The branch therefore accumulates the next version as a sequence of small reviewed changes instead of one large release merge. Every merge must leave `main` buildable and tested; unfinished behavior should remain unmerged or be protected by a disabled-by-default feature flag.

The previous release tag is the lower boundary of the next changelog. When the release pull request finally changes `RELEASE`, GitHub compares that previous tag with the release commit and discovers every merged pull request in between. Patch maintenance for older release lines would require explicit release branches and backports; the pre-1.0 project currently supports only the latest published line.

## Pipeline behavior

The `CI` workflow runs Go and frontend lint, tests, and production builds for `main` and pull requests. The separate `Release` workflow runs only for a direct push to `main` where `RELEASE` changed, or an explicit recovery dispatch naming an existing tag. It does not listen for tags, preventing a release loop. GitHub Actions concurrency serializes publication.

For a normal release, the workflow resolves the `main` commit that changed `RELEASE` as `source_sha` and derives the build timestamp from that commit. It performs the full Go, frontend, release, Compose, Markdown, and diff verification before publication. It then anchors `vVERSION` to `source_sha`, publishes only the immutable `VERSION` and `vVERSION` image tags, captures their digest, creates the GitHub Release against the same SHA, and finally promotes that exact digest to `latest` for a stable release. A prerelease never runs the `latest` promotion job. Every source-dependent checkout explicitly uses `source_sha`.

The tag is deliberately created after verification and before the first external artifact. If image publication fails, the tag records the only source permitted for recovery. Existing tags are accepted only when they already resolve to the requested source SHA; the workflow never moves or force-updates one. Existing version images are reused only when their OCI revision, version, and source-derived creation timestamp match. GitHub Release target and prerelease state are likewise verified before an existing release is treated as complete.

Partial failures are safe at each boundary: validation or verification publishes nothing; failure after tag anchoring recovers from that tag; failure after image publication reuses the verified digest and repairs the GitHub Release; failure after the GitHub Release promotes only that digest to `latest`. Repeating a completed recovery verifies and reuses immutable artifacts instead of rebuilding them.

Images published before OCI release-identity labels were introduced cannot be proven equivalent by automation. Recovery fails closed when such a legacy version image already exists: a maintainer must inspect its provenance and choose a separate, reviewed migration procedure. The release workflow never overwrites an unverifiable immutable tag.

## Release notes

The release workflow asks GitHub to generate notes for the exact range between the SemVer-previous tag and the immutable release commit. It enriches every discovered entry with GitHub PR metadata: number, title, author, head branch, labels, merge time, URL, and—only when other evidence is ambiguous—changed files. This preserves authoritative pull request links and GitHub's `New Contributors` section. The final comparison link is constructed explicitly as `previous_tag...current_tag`.

PGSentinel formats entries into Security, Features, Bug fixes, Performance, Documentation, Dependencies, Tests, Maintenance, and General Changes. Classification uses security and dependency evidence first, then conventional titles, known branch prefixes, and finally an all-documentation changed-file fallback. Entries are sorted by merge time (or PR number when merge time is unavailable), making regeneration byte-stable. Conventional prefixes are removed from display titles.

The workflow identifies the PR associated with `source_sha`, requires that it targets `main` and changes `RELEASE`, and excludes it automatically. Ambiguous associations fail closed; a genuine direct push may have no source PR. The `skip-changelog` label remains available for other intentional exclusions. `## Changelog (N)` is calculated only after both exclusions.

### Historical note regeneration

Maintainers can preview corrected notes for an existing immutable release without changing GitHub:

```bash
./scripts/regenerate-release-notes.sh v0.8.1
```

The tool verifies `tag -> commit -> RELEASE`, verifies the GitHub Release target, selects the true SemVer predecessor, regenerates and enriches notes, and prints a unified diff plus counts and category metadata. Dry-run is the default. Only an explicit `--apply` edits the public body; review the complete diff first. It never creates or moves tags and never rebuilds an image.

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

Stable `0.7.0` produces `ghcr.io/matta813/pgsentinel:0.7.0`, `:v0.7.0`, and `:latest`. Pre-release `1.0.0-rc.1` produces only `:1.0.0-rc.1` and `:v1.0.0-rc.1`; it never changes `latest`. Pin production to a fixed version.

Build arguments and OCI labels embed version, immutable source commit SHA, and that commit's timestamp. They are visible at `GET /api/v1/version` and in the sidebar. Recovery therefore reproduces the same application identity even after `main` advances. Local builds remain `dev`, `unknown`, `unknown`.

## Authentication and protection

The release job uses GitHub's short-lived `GITHUB_TOKEN`; no long-lived release secret is required. Its job-scoped permissions are limited to `contents: write` for the tag/release and `packages: write` for GHCR. All other workflows default to `contents: read`.

Recommended settings: protect `main` with a GitHub ruleset, require the CI checks and reviews, grant the release workflow write access through `GITHUB_TOKEN`, and keep GHCR linked to this repository. Dependabot updates remain manual.

## Troubleshooting

- **Invalid version:** run `./scripts/test-release.sh` and `./scripts/validate-release.sh RELEASE`.
- **Version went backwards:** choose a version greater than the newest release under SemVer precedence.
- **Tag exists:** never reuse versions. Normal retries and recovery proceed only when the tag already points to the resolved source SHA.
- **Registry denied:** ensure GitHub Actions has `packages: write` and the package is linked to this repository.
- **Release API denied:** ensure the workflow has `contents: write` and repository Actions are allowed write access.
- **No release jobs:** ensure this is a direct push to `main` and `RELEASE` actually changed.
- **Retry after infrastructure failure:** open **Actions → Release → Run workflow**, select `main`, and enter the exact existing tag, such as `v0.8.1`. Recovery resolves the tag commit, verifies its `RELEASE` value, checks any existing image and GitHub Release identity, and repairs only missing publication stages. It never uses current `main` as product source. Do not rerun an old job when its workflow definition itself was faulty.
