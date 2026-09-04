#!/bin/sh
set -eu
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
cat > "$temp_dir/metadata.json" <<'EOF'
{
  "repository": "matta813/PGSentinel", "version": "0.9.0", "tag": "v0.9.0",
  "previous_tag": "v0.9.0-rc.2",
  "source_pr": {"number": 150, "url": "https://github.com/matta813/PGSentinel/pull/150"},
  "generated_body": "## What's Changed\n\n## New Contributors\n* @alice made their first contribution in https://github.com/matta813/PGSentinel/pull/140\n\n**Full Changelog**: wrong",
  "prs": [
    {"number": 150, "title": "chore: release 0.9.0", "head_ref": "release/0.9.0", "labels": [], "author": "alice", "merged_at": "2026-01-10T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/150", "files": ["RELEASE"]},
    {"number": 149, "title": "Hidden two", "head_ref": "misc/two", "labels": ["skip-changelog"], "author": "alice", "merged_at": "2026-01-09T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/149", "files": []},
    {"number": 148, "title": "Hidden one", "head_ref": "misc/one", "labels": ["skip-changelog"], "author": "alice", "merged_at": "2026-01-08T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/148", "files": []},
    {"number": 147, "title": "Unknown change", "head_ref": "work/unknown", "labels": [], "author": "alice", "merged_at": "2026-01-07T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/147", "files": ["internal/a.go"]},
    {"number": 146, "title": "Restructure internals", "head_ref": "refactor/packages", "labels": [], "author": "alice", "merged_at": "2026-01-06T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/146", "files": []},
    {"number": 145, "title": "Bump module", "head_ref": "dependabot/go_modules/x", "labels": [], "author": "dependabot[bot]", "merged_at": "2026-01-05T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/145", "files": []},
    {"number": 144, "title": "Harden callbacks", "head_ref": "security/callbacks", "labels": [], "author": "alice", "merged_at": "2026-01-04T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/144", "files": []},
    {"number": 143, "title": "Roadmap refresh", "head_ref": "docs/roadmap", "labels": [], "author": "alice", "merged_at": "2026-01-03T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/143", "files": []},
    {"number": 142, "title": "Repair deletion", "head_ref": "fix/deletion", "labels": [], "author": "alice", "merged_at": "2026-01-02T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/142", "files": []},
    {"number": 141, "title": "Descriptive feature", "head_ref": "feat/intelligence", "labels": [], "author": "alice", "merged_at": "2026-01-01T12:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/141", "files": []},
    {"number": 140, "title": "feat(auth): add administrator login", "head_ref": "topic/auth", "labels": [], "author": "alice", "merged_at": "2026-01-01T00:00:00Z", "url": "https://github.com/matta813/PGSentinel/pull/140", "files": []}
  ]
}
EOF
script=$(dirname "$0")/release-notes.sh
"$script" "$temp_dir/metadata.json" --output "$temp_dir/one.md" --stats "$temp_dir/stats.json"
"$script" "$temp_dir/metadata.json" --output "$temp_dir/two.md"
cmp "$temp_dir/one.md" "$temp_dir/two.md"
grep -q '^## Changelog (8)$' "$temp_dir/one.md"
for heading in '🚀 Features' '🐛 Bug fixes' '📚 Documentation' '🔒 Security' '📦 Dependencies' '🧰 Maintenance' '📈 General Changes'; do
  grep -q "^### $heading$" "$temp_dir/one.md"
done
grep -q '^\* Add administrator login ' "$temp_dir/one.md"
grep -q '^\* Descriptive feature ' "$temp_dir/one.md"
grep -q '^## New Contributors$' "$temp_dir/one.md"
grep -q 'compare/v0.9.0-rc.2...v0.9.0$' "$temp_dir/one.md"
! grep -q 'release 0.9.0' "$temp_dir/one.md"
! grep -q 'Hidden one\|Hidden two' "$temp_dir/one.md"
[ "$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["count"])' "$temp_dir/stats.json")" = 8 ]
first=$(grep -n '\[PR #140\]' "$temp_dir/one.md" | cut -d: -f1)
last=$(grep -n '\[PR #147\]' "$temp_dir/one.md" | cut -d: -f1)
[ "$first" -lt "$last" ]
python3 -B - "$(dirname "$0")/format-release-notes.py" <<'PY'
import importlib.util
import sys
spec = importlib.util.spec_from_file_location("release_notes", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
expected = {"feat": "features", "fix": "fixes", "perf": "performance", "docs": "documentation",
            "test": "tests", "refactor": "maintenance", "chore": "maintenance", "ci": "maintenance", "build": "maintenance"}
for kind, category in expected.items():
    pr = {"title": f"{kind}(scope): useful change", "head_ref": "topic/x", "labels": [], "author": "alice", "files": []}
    assert module.classify(pr) == (category, "Useful change")
docs = {"title": "Clarify operations", "head_ref": "topic/help", "labels": [], "author": "alice", "files": ["README.md", "docs/releases.md"]}
assert module.classify(docs)[0] == "documentation"
PY
echo "release notes tests passed"
