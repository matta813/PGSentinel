#!/usr/bin/env python3
import argparse
import json
import re
from collections import OrderedDict
from pathlib import Path

CATEGORIES = OrderedDict([
    ("security", "🔒 Security"), ("features", "🚀 Features"),
    ("fixes", "🐛 Bug fixes"), ("performance", "⚡ Performance"),
    ("documentation", "📚 Documentation"), ("dependencies", "📦 Dependencies"),
    ("tests", "🧪 Tests"), ("maintenance", "🧰 Maintenance"),
    ("general", "📈 General Changes"),
])
CONVENTIONAL = re.compile(
    r"^(?P<kind>feat|fix|perf|docs|test|refactor|chore|ci|build)"
    r"(?:\((?P<scope>[^)]+)\))?(?:!)?:\s*(?P<title>.+)$", re.IGNORECASE,
)
BRANCH_CATEGORIES = (
    (("feat/", "feature/"), "features"), (("fix/", "bugfix/"), "fixes"),
    (("security/",), "security"), (("perf/",), "performance"),
    (("docs/",), "documentation"), (("test/",), "tests"),
    (("refactor/", "chore/", "ci/", "build/"), "maintenance"),
)


def normalized_labels(pr: dict) -> set[str]:
    return {str(label).strip().lower() for label in pr.get("labels", [])}


def is_dependency(pr: dict, labels: set[str]) -> bool:
    title, branch, author = pr["title"].lower(), pr.get("head_ref", "").lower(), pr.get("author", "").lower()
    return (title.startswith(("chore(deps):", "chore(deps-dev):"))
            or author == "dependabot[bot]" or branch.startswith("dependabot/")
            or any("depend" in label for label in labels))


def is_security(pr: dict, labels: set[str]) -> bool:
    title, branch = pr["title"].lower(), pr.get("head_ref", "").lower()
    return (title.startswith("fix(security):") or branch.startswith("security/")
            or any(label in {"security", "type: security", "changelog: security"} for label in labels))


def documentation_only(files: list[str]) -> bool:
    roots = {"readme.md", "security.md", "contributing.md", "code_of_conduct.md", "license"}
    return bool(files) and all(path.lower().startswith("docs/") or path.lower() in roots
                               or path.lower().endswith((".md", ".mdx", ".rst")) for path in files)


def classify(pr: dict) -> tuple[str, str]:
    title, labels = pr["title"].strip(), normalized_labels(pr)
    if is_security(pr, labels):
        category = "security"
    elif is_dependency(pr, labels):
        category = "dependencies"
    else:
        match = CONVENTIONAL.match(title)
        if match:
            category = {"feat": "features", "fix": "fixes", "perf": "performance",
                        "docs": "documentation", "test": "tests", "refactor": "maintenance",
                        "chore": "maintenance", "ci": "maintenance", "build": "maintenance"}[match.group("kind").lower()]
        else:
            branch, category = pr.get("head_ref", "").lower(), ""
            for prefixes, candidate in BRANCH_CATEGORIES:
                if branch.startswith(prefixes):
                    category = candidate
                    break
            if not category:
                category = "documentation" if documentation_only(pr.get("files", [])) else "general"
    match = CONVENTIONAL.match(title)
    clean = match.group("title").strip() if match else title
    if match and clean:
        clean = clean[0].upper() + clean[1:]
    return category, clean


def extract_new_contributors(body: str) -> list[str]:
    lines = body.splitlines()
    try:
        start = lines.index("## New Contributors")
    except ValueError:
        return []
    result = ["## New Contributors"]
    for line in lines[start + 1:]:
        if line.startswith("**Full Changelog**") or line.startswith("## "):
            break
        result.append(line)
    while result and not result[-1].strip():
        result.pop()
    return result if len(result) > 1 else []


def sorted_prs(prs: list[dict]) -> list[dict]:
    if all(pr.get("merged_at") for pr in prs):
        return sorted(prs, key=lambda pr: (pr["merged_at"], int(pr["number"])))
    return sorted(prs, key=lambda pr: int(pr["number"]))


def render(metadata: dict) -> tuple[str, dict]:
    version, tag = metadata["version"], metadata.get("tag", f"v{metadata['version']}")
    source_pr = metadata.get("source_pr")
    source_number = int(source_pr["number"]) if source_pr else None
    excluded, included = [], []
    for pr in metadata.get("prs", []):
        reason = None
        if source_number is not None and int(pr["number"]) == source_number:
            reason = "release-source"
        elif "skip-changelog" in normalized_labels(pr):
            reason = "skip-changelog"
        (excluded if reason else included).append(
            {"number": int(pr["number"]), "reason": reason} if reason else pr
        )

    sections, classified = {key: [] for key in CATEGORIES}, {}
    for pr in sorted_prs(included):
        category, title = classify(pr)
        classified[str(pr["number"])] = category
        sections[category].append(f"* {title} [PR #{pr['number']}]({pr['url']}), by @{pr['author']}")

    kind = "preview release" if "-" in version else "stable release"
    output = [f"# 🚀 PGSentinel {version}", "",
              f"We are pleased to announce PGSentinel {version}, the latest {kind} of PGSentinel!",
              "This release improves PostgreSQL monitoring while keeping recommendations evidence-driven and operator-controlled.",
              "Before upgrading, back up the PGSentinel data volume and keep the encryption key and administrator password available.",
              "", f"## Changelog ({len(included)})"]
    for category, heading in CATEGORIES.items():
        if sections[category]:
            output.extend(("", f"### {heading}", *sections[category]))
    contributors = extract_new_contributors(metadata.get("generated_body", ""))
    if contributors:
        output.extend(("", *contributors))
    previous_tag = metadata.get("previous_tag", "")
    if previous_tag:
        output.extend(("", f"**Full Changelog**: https://github.com/{metadata['repository']}/compare/{previous_tag}...{tag}"))
    output.append("")
    stats = {"count": len(included), "excluded": excluded, "source_pr": source_number,
             "categories": classified, "previous_tag": previous_tag}
    return "\n".join(output), stats


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("metadata", type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--stats", type=Path)
    args = parser.parse_args()
    body, stats = render(json.loads(args.metadata.read_text(encoding="utf-8")))
    if args.output:
        args.output.write_text(body, encoding="utf-8")
    else:
        print(body, end="")
    if args.stats:
        args.stats.write_text(json.dumps(stats, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
