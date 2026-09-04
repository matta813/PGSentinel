#!/bin/sh
set -eu
. "$(dirname "$0")/release-lib.sh"

usage() {
  echo "usage: release-metadata.sh normal SOURCE_SHA | recovery TAG" >&2
  exit 2
}

[ "$#" -eq 2 ] || usage
mode=$1
identity=$2

case "$mode" in
  normal)
    source_sha=$(git rev-parse --verify "$identity^{commit}")
    version=$(release_version_at_sha "$source_sha")
    tag="v$version"
    if git show-ref --verify --quiet "refs/tags/$tag"; then
      existing_sha=$(tag_commit_sha "$tag")
      if [ "$existing_sha" != "$source_sha" ]; then
        echo "Refusing to move $tag from $existing_sha to $source_sha" >&2
        exit 1
      fi
    else
      latest=$(latest_release_tag)
      if [ -n "$latest" ] && [ "$(compare_semver "$version" "${latest#v}")" -le 0 ]; then
        echo "Release version $version must be greater than $latest" >&2
        exit 1
      fi
    fi
    ;;
  recovery)
    tag=$identity
    version=${tag#v}
    [ "v$version" = "$tag" ] && validate_semver "$version" || {
      echo "Recovery requires a valid v-prefixed Semantic Version tag" >&2
      exit 1
    }
    source_sha=$(tag_commit_sha "$tag")
    source_version=$(release_version_at_sha "$source_sha")
    if [ "$source_version" != "$version" ]; then
      echo "Tag $tag points to RELEASE $source_version, expected $version" >&2
      exit 1
    fi
    ;;
  *) usage ;;
esac

previous_tag=$(previous_release_tag "$version")
source_timestamp=$(git show -s --format=%cI "$source_sha")
stable=true
if is_prerelease "$version"; then stable=false; fi

cat <<EOF
version=$version
tag=$tag
stable=$stable
source_sha=$source_sha
source_timestamp=$source_timestamp
previous_tag=$previous_tag
mode=$mode
EOF
