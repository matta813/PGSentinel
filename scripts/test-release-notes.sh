#!/bin/sh
set -eu
output=$("$(dirname "$0")/release-notes.sh" 0.1.0)
printf '%s' "$output" | grep -q '^# PGSentinel v0.1.0$'
printf '%s' "$output" | grep -q '^## Features$'
printf '%s' "$output" | grep -q -- '- add '
echo "release notes test passed"
