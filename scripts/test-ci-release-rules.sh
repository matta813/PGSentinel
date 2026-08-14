#!/bin/sh
set -eu
ci=.gitlab-ci.yml
grep -Fq 'if: $CI_COMMIT_TAG' "$ci"
grep -A1 -F 'if: $CI_COMMIT_TAG' "$ci" | grep -Fq 'when: never'
grep -Fq 'changes: [RELEASE]' "$ci"
grep -Fq 'CI_PIPELINE_SOURCE == "push" && $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH' "$ci"
grep -Fq 'resource_group: production-release' "$ci"
count=$(grep -cE '^[[:space:]]+- docker build ' "$ci")
[ "$count" -eq 1 ] || { echo "expected exactly one release-only docker build, got $count"; exit 1; }
echo "CI release rule tests passed"
