#!/bin/sh
set -eu
. "$(dirname "$0")/release-lib.sh"
usage() { echo "usage: regenerate-release-notes.sh TAG [--apply]" >&2; exit 2; }
[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage
tag=$1
apply=false
if [ "$#" -eq 2 ]; then [ "$2" = "--apply" ] || usage; apply=true; fi
version=${tag#v}
[ "v$version" = "$tag" ] && validate_semver "$version" || usage

repository=${GITHUB_REPOSITORY:-matta813/PGSentinel}
source_sha=$(tag_commit_sha "$tag")
tag_version=$(release_version_at_sha "$source_sha")
[ "$tag_version" = "$version" ] || { echo "Tag $tag contains RELEASE $tag_version" >&2; exit 1; }
previous_tag=$(previous_release_tag "$version")
release_json=$(gh api "repos/$repository/releases/tags/$tag")
release_target=$(printf '%s' "$release_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["target_commitish"])')
target_sha=$(git rev-parse --verify "$release_target^{commit}")
[ "$target_sha" = "$source_sha" ] || {
  echo "GitHub Release target $target_sha differs from tag $source_sha" >&2; exit 1;
}

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
RELEASE_NOTES_STATS="$temp_dir/stats.json" "$(dirname "$0")/generate-release-notes.sh" \
  "$repository" "$version" "$tag" "$source_sha" "$previous_tag" "$temp_dir/corrected.md"
printf 'Corrected release notes preview for %s (source %s, previous %s)\n' "$tag" "$source_sha" "${previous_tag:-none}"
printf '%s' "$release_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["body"])' > "$temp_dir/current.md"
diff -u "$temp_dir/current.md" "$temp_dir/corrected.md" || true
cat "$temp_dir/stats.json"
if [ "$apply" = true ]; then
  gh release edit "$tag" --repo "$repository" --notes-file "$temp_dir/corrected.md"
  echo "Updated GitHub Release $tag"
else
  echo "Dry run only; pass --apply to update the GitHub Release"
fi
