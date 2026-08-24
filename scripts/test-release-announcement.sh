#!/bin/sh
set -eu

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

cat > "$temp_dir/release-notes.md" <<'EOF'
# PGSentinel 0.6.0

## Features

- Add replication health findings by @alice in https://github.com/matta813/PGSentinel/pull/101
- Explain query impact score inputs

## New Contributors

- @alice made their first contribution
EOF

python3 "$(dirname "$0")/generate-release-announcement.py" \
  --version 0.6.0 \
  --title 'PGSentinel v0.6.0' \
  --notes "$temp_dir/release-notes.md" \
  --release-url https://github.com/matta813/PGSentinel/releases/tag/v0.6.0 \
  --output "$temp_dir/announcement.md"

grep -q '^# PGSentinel v0.6.0 Announcement$' "$temp_dir/announcement.md"
grep -q '^> Human review required\.' "$temp_dir/announcement.md"
grep -q '^- Add replication health findings$' "$temp_dir/announcement.md"
grep -q '^## Short$' "$temp_dir/announcement.md"
grep -q '^## Community$' "$temp_dir/announcement.md"
grep -q '^## GitHub Discussion$' "$temp_dir/announcement.md"
grep -q '^## Upgrade$' "$temp_dir/announcement.md"
grep -q 'releases/tag/v0.6.0' "$temp_dir/announcement.md"

if python3 "$(dirname "$0")/generate-release-announcement.py" --version latest --title latest --notes "$temp_dir/release-notes.md" --release-url https://github.com/matta813/PGSentinel/releases/tag/latest >/dev/null 2>&1; then
  echo "invalid versions must be rejected" >&2
  exit 1
fi

echo "release announcement test passed"
