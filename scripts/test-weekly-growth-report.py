#!/usr/bin/env python3

from datetime import datetime, timezone
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
from unittest.mock import patch

sys.dont_write_bytecode = True

script = Path(__file__).with_name("generate-weekly-growth-report.py")
spec = importlib.util.spec_from_file_location("weekly_growth", script)
module = importlib.util.module_from_spec(spec)
assert spec.loader
spec.loader.exec_module(module)


class FakeGitHub:
    def __init__(self, repository: str, token: str) -> None:
        assert repository == "matta813/PGSentinel"
        assert token == "test-token"

    def get(self, path: str, optional: bool = False):
        if path == "/repos/matta813/PGSentinel":
            return {"stargazers_count": 12, "forks_count": 2, "subscribers_count": 3}, {}
        if path.endswith("/traffic/views"):
            return None, {}
        if path.endswith("/traffic/clones"):
            return {"count": 7, "uniques": 4}, {}
        raise AssertionError(f"unexpected API path: {path}")

    def all(self, path: str):
        if "/releases" in path:
            return [{"name": "PGSentinel v0.5.0", "html_url": "https://example.test/release", "published_at": "2026-08-20T10:00:00Z"}]
        if "/contributors" in path:
            return [{"login": "one"}, {"login": "two"}, {"login": "three"}]
        if "/commits" in path:
            return [{"sha": str(index)} for index in range(5)]
        raise AssertionError(f"unexpected list path: {path}")

    def search_count(self, query: str) -> int:
        if "type:issue state:open" in query:
            return 4
        if "type:pr state:open" in query:
            return 1
        if "type:issue created:" in query:
            return 2
        if "is:merged" in query:
            return 3
        raise AssertionError(f"unexpected search: {query}")


with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    previous = root / "previous.json"
    report = root / "report.md"
    snapshot = root / "snapshot.json"
    previous.write_text(json.dumps({"repository": {"stars": 8, "forks": 1, "watchers": 3, "openIssues": 2, "openPullRequests": 1, "contributors": 2, "releases": 0}}), encoding="utf-8")
    argv = [str(script), "--repository", "matta813/PGSentinel", "--token", "test-token", "--previous", str(previous), "--report", str(report), "--snapshot", str(snapshot)]
    with patch.object(module, "GitHub", FakeGitHub), patch.object(sys, "argv", argv), patch.object(module, "datetime") as clock:
        clock.now.return_value = datetime(2026, 8, 31, 8, 0, tzinfo=timezone.utc)
        module.main()

    output = report.read_text(encoding="utf-8")
    assert "Period: 2026-08-24 → 2026-08-31" in output
    assert "Stars: 12 (+4)" in output
    assert "Merged pull requests: 3" in output
    assert "Views: Unavailable" in output
    assert "Clones: 7" in output
    assert "no value is estimated" in output
    data = json.loads(snapshot.read_text(encoding="utf-8"))
    assert data["repository"]["contributors"] == 3
    assert data["traffic"]["views"] is None

print("weekly growth report test passed")
