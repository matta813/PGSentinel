#!/bin/sh
set -eu
metadata=${1:?release metadata JSON required}
shift
script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
exec python3 "$script_dir/format-release-notes.py" "$metadata" "$@"
