#!/bin/sh
set -eu
. "$(dirname "$0")/release-lib.sh"

release_file=${1:-RELEASE}
version=$(trim_version "$release_file")
raw=$(cat "$release_file")

if [ "$raw" != "$version" ] || [ "$(wc -l < "$release_file" | tr -d ' ')" -ne 1 ]; then
  echo "RELEASE must contain exactly one version line without whitespace" >&2
  exit 1
fi
if ! validate_semver "$version"; then
  echo "Invalid Semantic Version in RELEASE: $version" >&2
  exit 1
fi

tag="v$version"
previous=${PREVIOUS_VERSION:-}
if [ -n "$previous" ]; then
  previous=${previous#v}
  if ! validate_semver "$previous"; then echo "Invalid previous version: $previous" >&2; exit 1; fi
  comparison=$(compare_semver "$version" "$previous")
  if [ "$comparison" -le 0 ]; then
    echo "Release version $version must be greater than $previous" >&2
    exit 1
  fi
fi

if [ -n "${CI_REPOSITORY_URL:-}" ] && git ls-remote --exit-code "$CI_REPOSITORY_URL" "refs/tags/$tag" >/dev/null 2>&1; then
  echo "Tag $tag already exists; refusing a duplicate release" >&2
  exit 1
fi

stable=true
if is_prerelease "$version"; then stable=false; fi
cat > release.env <<EOF
RELEASE_VERSION=$version
RELEASE_TAG=$tag
RELEASE_STABLE=$stable
EOF
printf 'Validated release %s (stable=%s)\n' "$tag" "$stable"
