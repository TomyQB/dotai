#!/usr/bin/env bash
# mem-cli SessionStart hook — runs once per Claude Code session.
#
# Finds the .agent-memory/index.md of the repo the user is working in and
# emits it as additionalContext via the hookSpecificOutput JSON envelope.
# Claude and any sub-agent spawned from this session can then Read the
# relevant doc on demand when the topic is within its watches scope.
#
# Never blocks the session. Missing python3, missing repo, or missing index
# is a silent no-op — the session starts as if the hook were not installed.

set -u

# python3 is used to parse the stdin JSON payload and to emit the output
# envelope. Without it we cannot safely JSON-escape the markdown content.
command -v python3 >/dev/null 2>&1 || exit 0

PAYLOAD="$(cat 2>/dev/null || true)"

# Claude Code delivers the current working directory in the payload. If the
# key is missing or the payload is unparseable, fall back to the shell's cwd.
CWD="$(printf '%s' "$PAYLOAD" | python3 -c 'import json,sys
try:
    print(json.loads(sys.stdin.read()).get("cwd", "") or "")
except Exception:
    pass
')"
[ -z "$CWD" ] && CWD="$(pwd)"

# Walk up from cwd until we find a .agent-memory/index.md or hit /.
DIR="$CWD"
INDEX=""
while [ "$DIR" != "/" ] && [ -n "$DIR" ]; do
  if [ -f "$DIR/.agent-memory/index.md" ]; then
    INDEX="$DIR/.agent-memory/index.md"
    break
  fi
  DIR="$(dirname "$DIR")"
done

[ -z "$INDEX" ] && exit 0
[ ! -s "$INDEX" ] && exit 0

python3 - "$INDEX" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    content = f.read()

header = (
    "This repo has living agent-memory documentation under .agent-memory/. "
    "The index below lists every doc and the source paths it watches. "
    "Before editing code that falls under a doc's scope, use the Read tool "
    "to open that doc for context. Sub-agents spawned from this session "
    "inherit the same expectation. Do not read every doc preemptively — "
    "consult on demand, only what matches the task at hand.\n\n"
    "=== .agent-memory/index.md ===\n\n"
)

envelope = {
    "hookSpecificOutput": {
        "hookEventName": "SessionStart",
        "additionalContext": header + content,
    }
}
print(json.dumps(envelope))
PY
