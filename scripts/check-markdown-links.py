#!/usr/bin/env python3
"""Check repository-local links and anchors in Markdown files."""

from pathlib import Path
import re
import sys
from urllib.parse import unquote


ROOT = Path(__file__).resolve().parent.parent
LINK = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
HEADING = re.compile(r"^#{1,6}\s+(.+?)\s*#*\s*$")


def slug(value: str) -> str:
    value = re.sub(r"<[^>]+>", "", value).strip().lower()
    value = re.sub(r"[^\w\- ]", "", value, flags=re.UNICODE)
    return re.sub(r"[ _]+", "-", value)


def anchors(path: Path) -> set[str]:
    found: set[str] = set()
    duplicates: dict[str, int] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        match = HEADING.match(line)
        if not match:
            continue
        base = slug(match.group(1))
        number = duplicates.get(base, 0)
        duplicates[base] = number + 1
        found.add(base if number == 0 else f"{base}-{number}")
    return found


def main() -> int:
    errors: list[str] = []
    markdown = [path for path in ROOT.rglob("*.md") if ".git" not in path.parts and "node_modules" not in path.parts]
    for source in markdown:
        text = source.read_text(encoding="utf-8")
        for raw in LINK.findall(text):
            target = raw.strip().split(maxsplit=1)[0].strip("<>")
            if not target or re.match(r"^(?:https?://|mailto:)", target):
                continue
            path_part, separator, fragment = target.partition("#")
            destination = source if not path_part else (source.parent / unquote(path_part)).resolve()
            try:
                destination.relative_to(ROOT)
            except ValueError:
                errors.append(f"{source.relative_to(ROOT)}: link escapes repository: {target}")
                continue
            if not destination.exists():
                errors.append(f"{source.relative_to(ROOT)}: missing target: {target}")
            elif separator and destination.suffix.lower() == ".md" and fragment not in anchors(destination):
                errors.append(f"{source.relative_to(ROOT)}: missing anchor: {target}")
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(f"checked local links in {len(markdown)} Markdown files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
