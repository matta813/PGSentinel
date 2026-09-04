#!/bin/sh
set -eu
script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
. "$script_dir/release-lib.sh"
failures=0
assert_valid(){ if ! validate_semver "$1"; then echo "expected valid: $1"; failures=$((failures+1)); fi; }
assert_invalid(){ if validate_semver "$1"; then echo "expected invalid: $1"; failures=$((failures+1)); fi; }
assert_compare(){ actual=$(compare_semver "$1" "$2"); if [ "$actual" != "$3" ]; then echo "compare $1 $2: expected $3, got $actual"; failures=$((failures+1)); fi; }
for value in 0.1.0 1.0.0 2.14.7 1.0.0-rc.1 1.0.0-beta.2 1.0.0-alpha.beta.1; do assert_valid "$value"; done
for value in v1.0.0 1.0 1 release-1.0.0 test 01.2.3 1.02.3 1.0.03 1.0.0-01 1.0.0-alpha.01; do assert_invalid "$value"; done
assert_compare 0.8.0 0.8.1 -1
assert_compare 0.9.0-beta.1 0.9.0-rc.1 -1
assert_compare 0.9.0-rc.1 0.9.0-rc.2 -1
assert_compare 0.9.0-rc.2 0.9.0 -1
assert_compare 0.9.0 0.9.1 -1
assert_compare 1.0.0-alpha.2 1.0.0-alpha.10 -1
assert_compare 1.0.0-alpha.1 1.0.0-alpha.beta -1
assert_compare 1.0.0-alpha-x 1.0.0-alpha-y -1
assert_compare 999999999999999999999.0.0 1000000000000000000000.0.0 -1
assert_compare 1.0.0 1.0.0-rc.1 1
if is_prerelease 1.0.0; then echo "stable classified as prerelease"; failures=$((failures+1)); fi
if ! is_prerelease 1.0.0-rc.1; then echo "prerelease classified as stable"; failures=$((failures+1)); fi

repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
release_version=$(cat "$repo_root/RELEASE")
compose_version=$(sed -n 's/.*PGSENTINEL_VERSION:-\([^}]*\)}.*/\1/p' "$repo_root/docker-compose.yml")
example_version=$(sed -n 's/^PGSENTINEL_VERSION=//p' "$repo_root/.env.example")
quickstart_version=$(sed -n 's/^[[:space:]]*image: ghcr.io\/matta813\/pgsentinel:\([^[:space:]]*\).*/\1/p' "$repo_root/docker-compose.quickstart.yml")
for actual in "$compose_version" "$example_version" "$quickstart_version"; do
  [ "$actual" = "$release_version" ] || { echo "published version $actual does not match RELEASE $release_version"; failures=$((failures+1)); }
done

temp_root=$(mktemp -d)
trap 'rm -rf "$temp_root"' EXIT
tag_repo=$temp_root/tags
git init -q "$tag_repo"
git -C "$tag_repo" config user.email test@example.com
git -C "$tag_repo" config user.name Test
for version in 0.8.0 0.9.0-beta.1 0.9.0-rc.1 0.9.0-rc.2 0.9.0; do
  printf '%s\n' "$version" > "$tag_repo/RELEASE"
  git -C "$tag_repo" add RELEASE
  git -C "$tag_repo" commit -qm "$version"
  git -C "$tag_repo" tag "v$version"
done
(
  cd "$tag_repo"
  [ "$(latest_release_tag)" = v0.9.0 ]
  [ "$(previous_release_tag 0.9.0)" = v0.9.0-rc.2 ]
  [ "$(previous_release_tag 0.9.1)" = v0.9.0 ]
  [ "$(tag_commit_sha v0.9.0)" = "$(git rev-parse 'v0.9.0^{}')" ]
) || { echo "SemVer tag selection failed"; failures=$((failures+1)); }

recovery_repo=$temp_root/recovery
git init -q "$recovery_repo"
git -C "$recovery_repo" config user.email test@example.com
git -C "$recovery_repo" config user.name Test
printf '0.8.1\n' > "$recovery_repo/RELEASE"
git -C "$recovery_repo" add RELEASE
git -C "$recovery_repo" commit -qm release
source_a=$(git -C "$recovery_repo" rev-parse HEAD)
git -C "$recovery_repo" tag v0.8.1
printf 'main advanced\n' > "$recovery_repo/main.txt"
git -C "$recovery_repo" add main.txt
git -C "$recovery_repo" commit -qm main-advanced
source_b=$(git -C "$recovery_repo" rev-parse HEAD)
metadata=$(cd "$recovery_repo" && "$script_dir/release-metadata.sh" recovery v0.8.1)
printf '%s\n' "$metadata" | grep -Fqx "source_sha=$source_a" || { echo "recovery did not use tag source"; failures=$((failures+1)); }
[ "$source_a" != "$source_b" ] || { echo "recovery fixture did not advance main"; failures=$((failures+1)); }
if (cd "$recovery_repo" && "$script_dir/release-metadata.sh" normal "$source_b" >/dev/null 2>&1); then
  echo "attempt to move an existing release tag succeeded"; failures=$((failures+1))
fi

mismatch_repo=$temp_root/mismatch
git init -q "$mismatch_repo"
git -C "$mismatch_repo" config user.email test@example.com
git -C "$mismatch_repo" config user.name Test
printf '0.8.0\n' > "$mismatch_repo/RELEASE"
git -C "$mismatch_repo" add RELEASE
git -C "$mismatch_repo" commit -qm mismatch
git -C "$mismatch_repo" tag v0.8.1
if (cd "$mismatch_repo" && "$script_dir/release-metadata.sh" recovery v0.8.1 >/dev/null 2>&1); then
  echo "tag/RELEASE mismatch succeeded"; failures=$((failures+1))
fi

[ "$failures" -eq 0 ] || exit 1
echo "release logic tests passed"
