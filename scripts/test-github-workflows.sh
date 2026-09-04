#!/bin/sh
set -eu

ci=.github/workflows/ci.yml
release=.github/workflows/release.yml
announcement=.github/workflows/release-announcement.yml
growth=.github/workflows/weekly-growth-report.yml
dependabot=.github/dependabot.yml
osv_config=osv-scanner.toml

test -f "$ci"
test -f "$release"
test -f "$announcement"
test -f "$growth"
test -f "$dependabot"
test -f "$osv_config"
grep -Fq 'paths: [RELEASE]' "$release"
grep -Fq 'packages: write' "$release"
grep -Fq 'docker/build-push-action@' "$release"
grep -Fq 'provenance: mode=max' "$release"
grep -Fq 'workflows: [Release]' "$announcement"
grep -Fq 'contents: read' "$announcement"
if grep -Eq 'github\.event\.workflow_run\.(head_sha|head_branch)' "$announcement"; then
  echo "release announcement must not check out workflow_run-controlled revisions" >&2
  exit 1
fi
grep -Fq 'cron: "17 8 * * 1"' "$growth"
grep -Fq 'actions: read' "$growth"
grep -Fq 'issues: read' "$growth"
grep -Fq 'pull-requests: read' "$growth"
if grep -Eq ':[[:space:]]*write([[:space:]]|$)' "$announcement" "$growth"; then
  echo "growth preparation workflows must remain read-only" >&2
  exit 1
fi
if grep -Eiq 'release create|discussion create|issue create|git push|pull request merge|auto.?merge|reddit|hacker news|linkedin|mastodon|bluesky|discord' "$announcement" "$growth"; then
  echo "growth preparation workflows must not publish, merge, or post externally" >&2
  exit 1
fi
if grep -Eq 'docker/(build-push|login)-action|docker build|docker push' "$ci"; then
  echo "normal CI must not build or push a container" >&2
  exit 1
fi
if grep -REq '^[[:space:]]*-[[:space:]]+uses:[[:space:]]+[^[:space:]]+@(v[0-9]+|main|master)([[:space:]]|$)' .github/workflows; then
  echo "GitHub Actions must be pinned to immutable commit SHAs" >&2
  exit 1
fi
if awk '/^FROM / && $2 !~ /@sha256:/ { found=1 } END { exit !found }' Dockerfile; then
  echo "Dockerfile base images must be pinned to immutable digests" >&2
  exit 1
fi
if ! grep -Eq '^# syntax=[^ ]+@sha256:[a-f0-9]{64}$' Dockerfile; then
  echo "Dockerfile frontend must be pinned to an immutable digest" >&2
  exit 1
fi
grep -Fq 'package-ecosystem: gomod' "$dependabot"
grep -Fq 'package-ecosystem: npm' "$dependabot"
grep -Fq 'package-ecosystem: docker' "$dependabot"
grep -Fq 'package-ecosystem: github-actions' "$dependabot"
for group in \
  go-patches go-minor go-major \
  frontend-runtime-patches frontend-development-patches \
  frontend-runtime-minor frontend-development-minor frontend-major \
  docker-patches docker-minor docker-major \
  github-actions-patches github-actions-minor github-actions-major; do
  grep -Fq "      $group:" "$dependabot"
done
for update_type in patch minor major; do
  grep -Fq "update-types: [$update_type]" "$dependabot"
done
if grep -Fq 'RELEASE' "$dependabot"; then
  echo "Dependabot must not manage RELEASE" >&2
  exit 1
fi
grep -Fq 'id = "GO-2026-5932"' "$osv_config"
grep -Fq 'openpgp package is not imported or linked' "$osv_config"
if grep -REq '"golang.org/x/crypto/openpgp([/"]|$)' --include='*.go' .; then
  echo "GO-2026-5932 exception is invalid when openpgp is imported" >&2
  exit 1
fi
echo "GitHub workflow tests passed"
