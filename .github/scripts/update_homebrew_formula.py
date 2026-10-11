#!/usr/bin/env python3
import sys
from pathlib import Path


def main() -> int:
    if len(sys.argv) != 4:
        print("usage: update_homebrew_formula.py <tag> <url> <sha256>", file=sys.stderr)
        return 2

    tag, url, sha256 = sys.argv[1:]
    if not tag:
        print("tag is required", file=sys.stderr)
        return 2
    if not url.startswith("https://"):
        print("url must be https", file=sys.stderr)
        return 2
    if len(sha256) != 64 or any(char not in "0123456789abcdef" for char in sha256.lower()):
        print("sha256 must be a 64-character hex digest", file=sys.stderr)
        return 2

    formula = Path("Formula/sealog.rb")
    text = formula.read_text()
    lines = text.splitlines()

    out: list[str] = []
    replaced = False
    index = 0
    while index < len(lines):
        line = lines[index]
        stripped = line.lstrip()

        if stripped.startswith("url "):
            indent = line[: len(line) - len(stripped)]
            out.append(f'{indent}url "{url}"')
            if index + 1 < len(lines) and lines[index + 1].lstrip().startswith("sha256 "):
                next_line = lines[index + 1]
                next_indent = next_line[: len(next_line) - len(next_line.lstrip())]
                out.append(f'{next_indent}sha256 "{sha256}"')
                index += 2
            else:
                out.append(f'{indent}sha256 "{sha256}"')
                index += 1
            replaced = True
            continue

        out.append(line)
        if not replaced and stripped.startswith("homepage "):
            indent = line[: len(line) - len(stripped)]
            out.append(f'{indent}url "{url}"')
            out.append(f'{indent}sha256 "{sha256}"')
            replaced = True
        index += 1

    if not replaced:
        print("could not find homepage or url stanza in Formula/sealog.rb", file=sys.stderr)
        return 1

    formula.write_text("\n".join(out) + "\n")
    print(f"updated Formula/sealog.rb for {tag}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
