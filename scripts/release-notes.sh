#!/bin/sh
set -eu

version=${1:?version required}
generated_notes=${2:?GitHub-generated notes file required}
script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)

exec python3 "$script_dir/format-release-notes.py" "$version" "$generated_notes"
