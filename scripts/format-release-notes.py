#!/usr/bin/env python3
import re
import sys
from collections import OrderedDict
from pathlib import Path


CATEGORIES = OrderedDict(
    [
        ("security", "🔒 Security"),
        ("features", "🚀 Features"),
        ("fixes", "🐛 Bug fixes"),
        ("performance", "⚡ Performance"),
        ("documentation", "📚 Documentation"),
        ("dependencies", "📦 Dependencies"),
        ("tests", "🧪 Tests"),
        ("maintenance", "🧰 Maintenance"),
        ("general", "📈 General Changes"),
    ]
)
CONVENTIONAL = re.compile(
    r"^(?P<kind>feat|fix|perf|docs|test|refactor|chore|ci|build)"
    r"(?:\((?P<scope>[^)]+)\))?(?:!)?:\s*(?P<title>.+)$",
    re.IGNORECASE,
)
ENTRY = re.compile(
    r"^\* (?P<title>.*?) by @(?P<author>\S+) in "
    r"(?P<reference>#\d+|https://github\.com/[^/]+/[^/]+/pull/\d+)$"
)


def classify(title: str) -> tuple[str, str]:
    match = CONVENTIONAL.match(title)
    if not match:
        return "general", title
    kind = match.group("kind").lower()
    scope = (match.group("scope") or "").lower()
    clean = match.group("title")
    if "security" in scope:
        category = "security"
    elif "dep" in scope:
        category = "dependencies"
    else:
        category = {
            "feat": "features",
            "fix": "fixes",
            "perf": "performance",
            "docs": "documentation",
            "test": "tests",
            "refactor": "maintenance",
            "chore": "maintenance",
            "ci": "maintenance",
            "build": "maintenance",
        }[kind]
    return category, clean[0].upper() + clean[1:] if clean else clean


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit("usage: format-release-notes.py VERSION GENERATED_NOTES")
    version = sys.argv[1]
    body = Path(sys.argv[2]).read_text(encoding="utf-8")
    sections = {key: [] for key in CATEGORIES}
    tail = []
    in_tail = False
    for line in body.splitlines():
        if line == "## What's Changed" or (line.startswith("### ") and not in_tail):
            continue
        if line == "## New Contributors" or line.startswith("**Full Changelog**"):
            in_tail = True
        if in_tail:
            tail.append(line)
            continue
        match = ENTRY.match(line)
        if match:
            category, title = classify(match.group("title"))
            reference = match.group("reference")
            if reference.startswith("https://"):
                number = reference.rsplit("/", 1)[-1]
                pr = f"[PR #{number}]({reference})"
            else:
                pr = f"PR {reference}"
            sections[category].append(f"* {title} {pr}, by @{match.group('author')}")

    count = sum(len(entries) for entries in sections.values())
    prerelease = "-" in version
    kind = "preview release" if prerelease else "stable release"
    print(f"# 🚀 PGSentinel {version}")
    print()
    print(f"We are pleased to announce PGSentinel {version}, the latest {kind} of PGSentinel!")
    print("This release improves PostgreSQL monitoring while keeping recommendations evidence-driven and operator-controlled.")
    print("Before upgrading, back up the PGSentinel data volume and keep the encryption key and administrator password available.")
    print()
    print(f"## Changelog ({count})")
    for category, heading in CATEGORIES.items():
        entries = sections[category]
        if entries:
            print()
            print(f"### {heading}")
            print("\n".join(entries))
    if tail:
        print()
        print("\n".join(tail).strip())


if __name__ == "__main__":
    main()
