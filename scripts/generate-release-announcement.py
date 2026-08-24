#!/usr/bin/env python3
"""Create a human-reviewable announcement draft from PGSentinel release notes."""

import argparse
from pathlib import Path
import re


def noteworthy_changes(notes: str, limit: int = 5) -> list[str]:
    changes: list[str] = []
    for line in notes.splitlines():
        item = re.match(r"^\s*[-*]\s+(.+)$", line)
        if not item:
            continue
        text = re.sub(r"\s+by\s+@[^ ]+(?:\s+in\s+https?://\S+)?$", "", item.group(1)).strip()
        if text and not text.startswith("@") and text not in changes:
            changes.append(text)
        if len(changes) == limit:
            break
    return changes


def render(version: str, title: str, release_url: str, notes: str) -> str:
    changes = noteworthy_changes(notes)
    change_lines = "\n".join(f"- {change}" for change in changes) or "- Review the full release notes for this version."
    return f"""# PGSentinel v{version} Announcement

> Human review required. Verify every claim and adapt the wording to the destination before publishing.

Release title: {title}

## Short

PGSentinel v{version} is available. It turns PostgreSQL telemetry into an operations inbox: problem, severity, evidence, impact, and next investigation. {release_url}

## Community

{title} is now available.

PGSentinel is a self-hosted PostgreSQL monitoring and health analysis tool built around an operations inbox rather than a wall of graphs. Findings explain what is wrong, why it matters, the supporting evidence, and what an operator should investigate next.

Highlights in this release:

{change_lines}

Read the release notes and upgrade guidance here: {release_url}

Feedback from PostgreSQL operators is welcome in GitHub Discussions. If you try the release, practical reports about setup, findings, and investigation usefulness are especially valuable.

## GitHub Discussion

{title} is available. This release includes the changes below. Please use this discussion for upgrade feedback and questions that may help other operators.

{change_lines}

Full release notes: {release_url}

## Key changes

{change_lines}

## Install

New local installation:

```bash
curl -fsSL https://raw.githubusercontent.com/matta813/PGSentinel/main/scripts/install-compose.sh | sh
```

## Upgrade

Read the release notes, back up `/data`, and follow the version-pinning instructions in the deployment guide. To pull the release image directly:

```bash
docker pull ghcr.io/matta813/pgsentinel:{version}
```

## Links

- Release: {release_url}
- Documentation: https://github.com/matta813/PGSentinel/tree/main/docs
- Discussions: https://github.com/matta813/PGSentinel/discussions
"""


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True, help="Release version without the v prefix")
    parser.add_argument("--title", required=True, help="Title from the GitHub Release")
    parser.add_argument("--notes", required=True, type=Path, help="Markdown release notes")
    parser.add_argument("--release-url", required=True, help="Canonical GitHub release URL")
    parser.add_argument("--output", type=Path, help="Write the draft here instead of stdout")
    args = parser.parse_args()

    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", args.version):
        parser.error("--version must be a Semantic Version without a v prefix")
    if not re.fullmatch(r"https://github\.com/matta813/PGSentinel/releases/tag/v[^\s]+", args.release_url):
        parser.error("--release-url must be a canonical PGSentinel GitHub release URL")

    output = render(args.version, args.title.strip(), args.release_url, args.notes.read_text(encoding="utf-8"))
    if args.output:
        args.output.write_text(output, encoding="utf-8")
    else:
        print(output, end="")


if __name__ == "__main__":
    main()
