#!/bin/sh
set -eu

ci=.github/workflows/ci.yml
release=.github/workflows/release.yml
dependabot=.github/dependabot.yml
osv_config=osv-scanner.toml

test -f "$ci"
test -f "$release"
test -f "$dependabot"
test -f "$osv_config"
grep -Fq 'paths: [RELEASE]' "$release"
grep -Fq 'packages: write' "$release"
grep -Fq 'docker/build-push-action@' "$release"
grep -Fq 'provenance: mode=max' "$release"
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
