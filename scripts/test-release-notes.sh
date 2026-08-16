#!/bin/sh
set -eu

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
cat > "$temp_dir/generated.md" <<'EOF'
## What's Changed
### Changes
* feat(auth): add administrator login by @alice in https://github.com/matta813/PGSentinel/pull/21
* fix(security): block private webhook targets by @bob in https://github.com/matta813/PGSentinel/pull/22
* chore(deps): bump example dependency by @dependabot[bot] in https://github.com/matta813/PGSentinel/pull/23
* improve operator guidance by @carol in #24

## New Contributors
* @alice made their first contribution in https://github.com/matta813/PGSentinel/pull/21

**Full Changelog**: https://github.com/matta813/PGSentinel/compare/v0.2.0...v0.3.0
EOF

"$(dirname "$0")/release-notes.sh" 0.3.0 "$temp_dir/generated.md" > "$temp_dir/output.md"
grep -q '^# 🚀 PGSentinel 0.3.0$' "$temp_dir/output.md"
grep -q '^## Changelog (4)$' "$temp_dir/output.md"
grep -q '^### 🚀 Features$' "$temp_dir/output.md"
grep -q '^\* Add administrator login \[PR #21\](https://github.com/matta813/PGSentinel/pull/21), by @alice$' "$temp_dir/output.md"
grep -q '^### 🔒 Security$' "$temp_dir/output.md"
grep -q '^### 📦 Dependencies$' "$temp_dir/output.md"
grep -q '^### 📈 General Changes$' "$temp_dir/output.md"
grep -q '^## New Contributors$' "$temp_dir/output.md"
grep -q '^\* @alice made their first contribution in https://github.com/matta813/PGSentinel/pull/21$' "$temp_dir/output.md"
grep -q '^\*\*Full Changelog\*\*:' "$temp_dir/output.md"
echo "release notes test passed"
