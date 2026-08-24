#!/usr/bin/env python3
"""Collect GitHub repository metrics and generate a factual weekly report."""

import argparse
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import re
import sys
from urllib.error import HTTPError
from urllib.parse import quote
from urllib.request import Request, urlopen


API = "https://api.github.com"


class GitHub:
    def __init__(self, repository: str, token: str) -> None:
        if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
            raise ValueError("repository must use owner/name format")
        self.repository = repository
        self.headers = {
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "PGSentinel-growth-report",
        }

    def get(self, path: str, optional: bool = False):
        request = Request(f"{API}{path}", headers=self.headers)
        try:
            with urlopen(request, timeout=30) as response:
                return json.load(response), response.headers
        except HTTPError as error:
            if optional and error.code in (403, 404):
                print(f"optional GitHub metric unavailable: {path} returned {error.code}", file=sys.stderr)
                return None, error.headers
            raise

    def all(self, path: str) -> list[dict]:
        separator = "&" if "?" in path else "?"
        page = 1
        results: list[dict] = []
        while True:
            data, _ = self.get(f"{path}{separator}per_page=100&page={page}")
            if not isinstance(data, list):
                raise ValueError(f"expected a list from GitHub API path {path}")
            results.extend(data)
            if len(data) < 100:
                return results
            page += 1

    def search_count(self, query: str) -> int:
        data, _ = self.get(f"/search/issues?q={quote(query)}&per_page=1")
        return int(data["total_count"])


def delta(current: int, previous: dict | None, key: str) -> str:
    if not previous or key not in previous.get("repository", {}):
        return ""
    change = current - int(previous["repository"][key])
    return f" ({change:+d})"


def metric(value: int | None) -> str:
    return str(value) if value is not None else "Unavailable"


def load_previous(path: Path | None) -> dict | None:
    if not path or not path.exists():
        return None
    return json.loads(path.read_text(encoding="utf-8"))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", default=os.environ.get("GITHUB_REPOSITORY"), help="GitHub owner/name")
    parser.add_argument("--token", default=os.environ.get("GITHUB_TOKEN"), help="GitHub token; defaults to GITHUB_TOKEN")
    parser.add_argument("--previous", type=Path, help="Previous snapshot JSON, when available")
    parser.add_argument("--report", required=True, type=Path, help="Markdown report output")
    parser.add_argument("--snapshot", required=True, type=Path, help="Current JSON snapshot output")
    args = parser.parse_args()
    if not args.repository or not args.token:
        parser.error("--repository and --token (or GITHUB_REPOSITORY/GITHUB_TOKEN) are required")

    client = GitHub(args.repository, args.token)
    now = datetime.now(timezone.utc)
    since = now - timedelta(days=7)
    since_iso = since.strftime("%Y-%m-%dT%H:%M:%SZ")
    since_day = since.strftime("%Y-%m-%d")
    until_day = now.strftime("%Y-%m-%d")
    repository, _ = client.get(f"/repos/{args.repository}")
    releases = client.all(f"/repos/{args.repository}/releases")
    contributors = client.all(f"/repos/{args.repository}/contributors?anon=1")
    commits = client.all(f"/repos/{args.repository}/commits?since={quote(since_iso)}")
    open_issues = client.search_count(f"repo:{args.repository} type:issue state:open")
    open_prs = client.search_count(f"repo:{args.repository} type:pr state:open")
    new_issues = client.search_count(f"repo:{args.repository} type:issue created:{since_day}..{until_day}")
    merged_prs = client.search_count(f"repo:{args.repository} type:pr is:merged merged:{since_day}..{until_day}")

    views, _ = client.get(f"/repos/{args.repository}/traffic/views", optional=True)
    clones, _ = client.get(f"/repos/{args.repository}/traffic/clones", optional=True)
    traffic = {
        "views": int(views["count"]) if views else None,
        "uniqueVisitors": int(views["uniques"]) if views else None,
        "clones": int(clones["count"]) if clones else None,
        "uniqueCloners": int(clones["uniques"]) if clones else None,
    }
    current = {
        "collectedAt": now.isoformat(),
        "periodStart": since.isoformat(),
        "repository": {
            "stars": int(repository["stargazers_count"]),
            "forks": int(repository["forks_count"]),
            "watchers": int(repository["subscribers_count"]),
            "openIssues": open_issues,
            "openPullRequests": open_prs,
            "contributors": len(contributors),
            "releases": len(releases),
        },
        "development": {"newIssues": new_issues, "mergedPullRequests": merged_prs, "commits": len(commits)},
        "traffic": traffic,
        "latestRelease": {"name": releases[0]["name"], "url": releases[0]["html_url"], "publishedAt": releases[0]["published_at"]} if releases else None,
    }
    previous = load_previous(args.previous)
    totals = current["repository"]
    latest = current["latestRelease"]
    highlights = [f"{merged_prs} pull request{'s were' if merged_prs != 1 else ' was'} merged during the period.", f"{new_issues} new issue{'s were' if new_issues != 1 else ' was'} opened during the period."]
    if latest:
        highlights.append(f"Latest release: [{latest['name']}]({latest['url']}) ({latest['publishedAt'][:10]}).")
    actions = []
    if new_issues:
        actions.append("Triage the newly opened issues and move support questions to Discussions where appropriate.")
    if merged_prs:
        actions.append("Review merged changes for the next release notes and technical content opportunities.")
    if not actions:
        actions.append("Review README onboarding and open Discussions for recurring points of friction.")
    actions.append("Choose at most one useful technical lesson or product update to share manually this week.")

    report = f"""# PGSentinel Weekly Growth Report

Period: {since_day} → {until_day}

## Repository

- Stars: {totals['stars']}{delta(totals['stars'], previous, 'stars')}
- Forks: {totals['forks']}{delta(totals['forks'], previous, 'forks')}
- Watchers: {totals['watchers']}{delta(totals['watchers'], previous, 'watchers')}
- Open issues: {totals['openIssues']}{delta(totals['openIssues'], previous, 'openIssues')}
- Open pull requests: {totals['openPullRequests']}{delta(totals['openPullRequests'], previous, 'openPullRequests')}
- Contributors: {totals['contributors']}{delta(totals['contributors'], previous, 'contributors')}

## Development

- Merged pull requests: {merged_prs}
- Commits: {len(commits)}
- Releases (total): {len(releases)}{delta(len(releases), previous, 'releases')}

## Traffic

- Views: {metric(traffic['views'])}
- Unique visitors: {metric(traffic['uniqueVisitors'])}
- Clones: {metric(traffic['clones'])}
- Unique cloners: {metric(traffic['uniqueCloners'])}

Traffic covers GitHub's available recent window, not necessarily the report period. `Unavailable` means the token lacks repository traffic access; no value is estimated.

## Highlights

{chr(10).join(f'- {item}' for item in highlights)}

## Suggested next actions

{chr(10).join(f'{index}. {item}' for index, item in enumerate(actions, 1))}
"""
    args.report.write_text(report, encoding="utf-8")
    args.snapshot.write_text(json.dumps(current, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
