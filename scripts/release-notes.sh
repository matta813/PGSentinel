#!/bin/sh
set -eu
version=${1:?version required}
previous_tag=${2:-}
range=HEAD
[ -z "$previous_tag" ] || range="$previous_tag..HEAD"

notes_for() {
  pattern=$1
  git log "$range" --pretty='format:%s' --no-merges | grep -E "$pattern" | sed -E 's/^[a-z]+(\([^)]*\))?!?:[[:space:]]*/- /' || true
}
section() {
  title=$1 pattern=$2
  content=$(notes_for "$pattern")
  [ -z "$content" ] || printf '\n## %s\n\n%s\n' "$title" "$content"
}

printf '# PGSentinel v%s\n' "$version"
section Features '^feat(\([^)]*\))?!?:'
section Fixes '^fix(\([^)]*\))?!?:'
section Performance '^perf(\([^)]*\))?!?:'
section Documentation '^docs(\([^)]*\))?!?:'
section 'Tests and maintenance' '^(test|refactor|chore|ci)(\([^)]*\))?!?:'
other=$(git log "$range" --pretty='format:%s' --no-merges | grep -Ev '^(feat|fix|perf|docs|test|refactor|chore|ci)(\([^)]*\))?!?:' | sed 's/^/- /' || true)
[ -z "$other" ] || printf '\n## Other changes\n\n%s\n' "$other"
