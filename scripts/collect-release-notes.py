#!/usr/bin/env python3
import argparse
import json
import re
import subprocess
from pathlib import Path

CONVENTIONAL = re.compile(r"^(feat|fix|perf|docs|test|refactor|chore|ci|build)(\([^)]+\))?!?:\s", re.I)
KNOWN_BRANCHES = ("feat/", "feature/", "fix/", "bugfix/", "security/", "perf/", "docs/", "test/", "refactor/", "chore/", "ci/", "build/", "dependabot/")


def gh(*args: str) -> object:
    result = subprocess.run(["gh", "api", *args], text=True, check=True, capture_output=True)
    return json.loads(result.stdout)


def gh_paginated(endpoint: str) -> list[dict]:
    result = subprocess.run(
        ["gh", "api", "--paginate", "--slurp", endpoint],
        text=True, check=True, capture_output=True,
    )
    return [item for page in json.loads(result.stdout) for item in page]


def pr_metadata(repository: str, number: int) -> dict:
    pr = gh(f"repos/{repository}/pulls/{number}")
    labels = [label["name"] for label in pr["labels"]]
    title, branch, author = pr["title"], pr["head"]["ref"], pr["user"]["login"]
    recognized = (title.lower().startswith(("fix(security):", "chore(deps):", "chore(deps-dev):"))
                  or branch.lower().startswith(("security/", "dependabot/"))
                  or author.lower() == "dependabot[bot]" or CONVENTIONAL.match(title)
                  or branch.lower().startswith(KNOWN_BRANCHES)
                  or any("depend" in label.lower() or label.lower() in {"security", "type: security", "changelog: security"} for label in labels))
    files = [] if recognized else [item["filename"] for item in gh_paginated(f"repos/{repository}/pulls/{number}/files?per_page=100")]
    return {"number": number, "title": title, "author": author, "head_ref": branch,
            "labels": labels, "merged_at": pr["merged_at"], "url": pr["html_url"], "files": files}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repository", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--source-sha", required=True)
    parser.add_argument("--previous-tag", default="")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    request = ["--method", "POST", f"repos/{args.repository}/releases/generate-notes",
               "-f", f"tag_name={args.tag}", "-f", f"target_commitish={args.source_sha}"]
    if args.previous_tag:
        request.extend(("-f", f"previous_tag_name={args.previous_tag}"))
    generated = gh(*request)
    numbers = sorted({int(value) for value in re.findall(r"/pull/(\d+)", generated["body"])})

    associated = gh("-H", "Accept: application/vnd.github+json",
                    f"repos/{args.repository}/commits/{args.source_sha}/pulls")
    candidates = []
    producing_prs = []
    for candidate in associated:
        if candidate["merged_at"] is None or candidate["base"]["ref"] != "main":
            continue
        if candidate.get("merge_commit_sha") != args.source_sha:
            continue
        producing_prs.append(candidate)
        files = gh_paginated(f"repos/{args.repository}/pulls/{candidate['number']}/files?per_page=100")
        if any(item["filename"] == "RELEASE" for item in files):
            candidates.append(candidate)
    if len(candidates) > 1:
        raise SystemExit("release source PR is ambiguous")
    if producing_prs and not candidates:
        raise SystemExit("source commit has associated PRs but none changes RELEASE")
    source_pr = None
    if candidates:
        source_pr = {"number": candidates[0]["number"], "url": candidates[0]["html_url"]}
        if candidates[0]["number"] not in numbers:
            numbers.append(candidates[0]["number"])

    metadata = {"repository": args.repository, "version": args.version, "tag": args.tag,
                "source_sha": args.source_sha, "previous_tag": args.previous_tag,
                "source_pr": source_pr, "generated_body": generated["body"],
                "prs": [pr_metadata(args.repository, number) for number in sorted(numbers)]}
    args.output.write_text(json.dumps(metadata, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
