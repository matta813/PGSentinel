# Automated releases

## Source of truth

`RELEASE` contains exactly one newline-terminated Semantic Version without `v`, comments or whitespace. Stable versions use `MAJOR.MINOR.PATCH`; pre-releases such as `1.0.0-rc.1` are supported. Validation rejects prefixes, incomplete versions, leading-zero numeric components, duplicate tags, unchanged effective values, and versions whose SemVer precedence is not greater than the latest `v*` tag.

Create a release by changing the value and pushing the commit to `main`:

```bash
printf '0.4.2\n' > RELEASE
git add RELEASE
git commit -m 'chore: release 0.4.2'
git push origin main
```

## Pipeline behavior

Normal pipelines run Go and frontend lint, tests, and production builds. `release-validation` and `publish-release` exist only for a direct push to the default branch where `RELEASE` changed. Tag pipelines are disabled at workflow level, preventing a release loop. `resource_group: production-release` serializes publication.

Publication order is image build, version image push, `v` image push, optional `latest` push, then GitLab release/tag creation. `glab release create` creates the tag from the exact `CI_COMMIT_SHA`; when a release exists it updates it, making the final operation idempotent. Image tags are safe to repush. Validation normally stops duplicate tags before expensive work.

## Image tags

Stable `0.4.2` produces `$CI_REGISTRY_IMAGE:0.4.2`, `$CI_REGISTRY_IMAGE:v0.4.2`, and `$CI_REGISTRY_IMAGE:latest`. Pre-release `1.0.0-rc.1` produces only `:1.0.0-rc.1` and `:v1.0.0-rc.1`; it never changes `latest`. Pin production to a fixed version.

Build arguments embed version, commit SHA and UTC build time. They are visible at `GET /api/v1/version` and in the sidebar. Local builds remain `dev`, `unknown`, `unknown`.

## Authentication and protection

Registry login uses `CI_REGISTRY_USER` and `CI_REGISTRY_PASSWORD`. Release/tag creation uses `CI_JOB_TOKEN` through `GLAB_ENABLE_CI_AUTOLOGIN=true`; current `glab` sends the Releases API-compatible `JOB-TOKEN` header. Do not assign `CI_JOB_TOKEN` to `GITLAB_TOKEN`.

If the GitLab instance disallows this job-token operation, create a masked, protected project/group access token named `GITLAB_TOKEN` with `api` scope and at least Developer role. `glab` automatically uses it. Never commit it.

Recommended settings: protect `main`, protect `v*` tags while allowing release automation, keep optional variables protected/masked, and ensure the runner can push to the Container Registry.

## Troubleshooting

- **Invalid version:** run `./scripts/test-release.sh` and `./scripts/validate-release.sh RELEASE`.
- **Version went backwards:** choose a version greater than the newest release under SemVer precedence.
- **Tag exists:** never reuse versions. For a partial publication, verify the tag target before repairing the release record.
- **Registry denied:** verify Registry enablement and standard CI registry variables.
- **Release API 404:** enable job-token Releases API access or add protected `GITLAB_TOKEN`.
- **No release jobs:** ensure this is a direct push to `main` and `RELEASE` actually changed.
- **Retry after infrastructure failure:** run a new pipeline on `main` with `RETRY_RELEASE=true`, either in GitLab or with `glab ci run -b main --variables RETRY_RELEASE:true`. This uses the current CI definition, revalidates the version/tag, and safely republishes image tags before creating the GitLab Release. Do not retry an old job when its CI configuration itself was faulty.
