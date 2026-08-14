#!/bin/sh
set -eu

ci=.github/workflows/ci.yml
release=.github/workflows/release.yml
dependabot=.github/dependabot.yml

test -f "$ci"
test -f "$release"
test -f "$dependabot"
grep -Fq 'paths: [RELEASE]' "$release"
grep -Fq 'packages: write' "$release"
grep -Fq 'docker/build-push-action@' "$release"
if grep -Eq 'docker/(build-push|login)-action|docker build|docker push' "$ci"; then
  echo "normal CI must not build or push a container" >&2
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
echo "GitHub workflow tests passed"
