#!/bin/sh
set -eu
. "$(dirname "$0")/release-lib.sh"
failures=0
assert_valid(){ if ! validate_semver "$1"; then echo "expected valid: $1"; failures=$((failures+1)); fi; }
assert_invalid(){ if validate_semver "$1"; then echo "expected invalid: $1"; failures=$((failures+1)); fi; }
assert_compare(){ actual=$(compare_semver "$1" "$2"); if [ "$actual" != "$3" ]; then echo "compare $1 $2: expected $3, got $actual"; failures=$((failures+1)); fi; }
for value in 0.1.0 1.0.0 2.14.7 1.0.0-rc.1 1.0.0-beta.2; do assert_valid "$value"; done
for value in v1.0.0 1.0 1 release-1.0.0 test 01.2.3 1.02.3; do assert_invalid "$value"; done
assert_compare 1.4.3 1.4.2 1
assert_compare 1.3.9 1.4.2 -1
assert_compare 1.0.0 1.0.0-rc.1 1
assert_compare 1.0.0-rc.2 1.0.0-rc.1 1
assert_compare 1.0.0-beta.2 1.0.0-rc.1 -1
if is_prerelease 1.0.0; then echo "stable classified as prerelease"; failures=$((failures+1)); fi
if ! is_prerelease 1.0.0-rc.1; then echo "prerelease classified as stable"; failures=$((failures+1)); fi

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
release_version=$(cat "$repo_root/RELEASE")
compose_version=$(sed -n 's/.*PGSENTINEL_VERSION:-\([^}]*\)}.*/\1/p' "$repo_root/docker-compose.yml")
example_version=$(sed -n 's/^PGSENTINEL_VERSION=//p' "$repo_root/.env.example")
quickstart_version=$(sed -n 's/^[[:space:]]*image: ghcr.io\/matta813\/pgsentinel:\([^[:space:]]*\).*/\1/p' "$repo_root/docker-compose.quickstart.yml")
if [ "$compose_version" != "$release_version" ]; then
  echo "docker-compose.yml version $compose_version does not match RELEASE $release_version"
  failures=$((failures+1))
fi
if [ "$example_version" != "$release_version" ]; then
  echo ".env.example version $example_version does not match RELEASE $release_version"
  failures=$((failures+1))
fi
if [ "$quickstart_version" != "$release_version" ]; then
  echo "docker-compose.quickstart.yml version $quickstart_version does not match RELEASE $release_version"
  failures=$((failures+1))
fi
[ "$failures" -eq 0 ] || exit 1
echo "release logic tests passed"
