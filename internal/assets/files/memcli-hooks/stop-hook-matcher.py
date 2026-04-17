"""Classify changed source files against the `watches` frontmatter of every
.md under .agent-memory/.

Reads changed paths (one per line, leading/trailing whitespace tolerated)
from stdin and prints one line per input:

    STALE <doc-path>       — path is covered by this doc's watches
    NEW   <orphan-path>    — path does not match any doc's watches

Orphans are reported once per path. A path covered by multiple docs produces
one STALE line per matching doc (the merge step in stop-hook.sh deduplicates).

Why this exists as a Python module instead of staying inline in bash: bash's
`case $glob` only matches single-segment globs, so `src/foo/**` could never
recurse past one level. fnmatch in the stdlib does not support `**` either,
and pathlib.Match only grew it in 3.13. Keeping a small custom glob
translator keeps us compatible with Python 3.8+ shipped on every supported
platform and lets us encode the `path/` → `path/**` convention used by our
docs.
"""

import os
import re
import sys


def translate(glob: str):
    """Translate a glob (possibly containing `**`) into a compiled regex.

    Convention: a pattern without wildcards that ends in `/` is treated as
    "everything under this folder" (equivalent to `<path>**`). A pattern
    without wildcards that does NOT end in `/` is a literal file match.
    """
    has_wildcard = any(ch in glob for ch in "*?[")
    if not has_wildcard and glob.endswith("/"):
        glob = glob + "**"
    i, n = 0, len(glob)
    out = ["^"]
    while i < n:
        c = glob[i]
        if c == "*":
            if i + 1 < n and glob[i + 1] == "*":
                out.append(".*")
                i += 2
                if i < n and glob[i] == "/":
                    i += 1
            else:
                out.append("[^/]*")
                i += 1
        elif c == "?":
            out.append("[^/]")
            i += 1
        elif c in ".+^$()|{}\\":
            out.append("\\" + c)
            i += 1
        elif c == "[":
            j = glob.find("]", i + 1)
            if j == -1:
                out.append("\\[")
                i += 1
            else:
                out.append(glob[i : j + 1])
                i = j + 1
        else:
            out.append(c)
            i += 1
    out.append("$")
    return re.compile("".join(out))


def parse_watches(doc_path):
    try:
        with open(doc_path, "r", encoding="utf-8") as f:
            content = f.read()
    except OSError:
        return []
    m = re.match(r"^---\n(.*?)\n---", content, re.DOTALL)
    if not m:
        return []
    fm = m.group(1)
    watches = []
    in_watches = False
    for line in fm.splitlines():
        if re.match(r"^watches:\s*$", line):
            in_watches = True
            continue
        if in_watches:
            m2 = re.match(r"^\s*-\s*(.+?)\s*$", line)
            if m2:
                w = m2.group(1).strip('"').strip("'")
                watches.append(w)
            elif re.match(r"^\S", line):
                in_watches = False
    return watches


def main(mem_dir):
    docs = []
    for root, _, files in os.walk(mem_dir):
        for fn in files:
            if not fn.endswith(".md"):
                continue
            full = os.path.join(root, fn)
            ws = parse_watches(full)
            if not ws:
                continue
            docs.append((full, [translate(w) for w in ws]))

    # .strip() here is the single source of truth for leading/trailing
    # whitespace — downstream awk in stop-hook.sh assumes clean paths.
    changed = [l.strip() for l in sys.stdin.read().splitlines() if l.strip()]

    for f in changed:
        matched = [d for d, patterns in docs if any(p.match(f) for p in patterns)]
        if matched:
            for d in matched:
                print("STALE", d)
        else:
            print("NEW", f)


if __name__ == "__main__":
    main(sys.argv[1])
