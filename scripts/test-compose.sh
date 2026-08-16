#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
export PGSENTINEL_ENCRYPTION_KEY=test-encryption-key
export PGSENTINEL_ADMIN_PASSWORD=test-administrator-password

for file in docker-compose.yml docker-compose.quickstart.yml; do
  docker compose -f "$repo_root/$file" config --format json | python3 -c '
import json, sys
config = json.load(sys.stdin)
actual = config["services"]["pgsentinel"].get("tmpfs")
expected = ["/tmp:rw,noexec,nosuid,nodev,size=64m"]
if actual != expected:
    raise SystemExit(f"invalid tmpfs configuration: {actual!r}")
'
done

echo "Compose configuration tests passed"
