#!/bin/sh
set -eu
repository=${1:?repository required}
version=${2:?version required}
tag=${3:?tag required}
source_sha=${4:?source SHA required}
previous_tag=${5:-}
output=${6:-release-notes.md}
script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
metadata=$(mktemp)
stats=${RELEASE_NOTES_STATS:-release-notes-stats.json}
trap 'rm -f "$metadata"' EXIT
python3 "$script_dir/collect-release-notes.py" --repository "$repository" --version "$version" \
  --tag "$tag" --source-sha "$source_sha" --previous-tag "$previous_tag" --output "$metadata"
python3 "$script_dir/format-release-notes.py" "$metadata" --output "$output" --stats "$stats"
