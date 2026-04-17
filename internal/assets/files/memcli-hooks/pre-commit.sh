#!/usr/bin/env bash
# mem-cli git pre-commit hook
# Blocks the commit if .agent-memory/.stale is non-empty.
# .stale has two kinds of entries:
#   path/to/doc.md          — existing doc to regenerate
#   [NEW] path/to/source    — orphaned source needing a brand-new doc
# Tells the user to run /memcli-update inside Claude Code to handle both.

set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
STALE_FILE="$REPO_ROOT/.agent-memory/.stale"

[ -f "$STALE_FILE" ] || exit 0

ENTRIES="$(sed '/^$/d' "$STALE_FILE")"
[ -z "$ENTRIES" ] && exit 0

STALE_LINES="$(printf '%s\n' "$ENTRIES" | grep -v '^\[NEW\] \S' || true)"
NEW_LINES="$(printf '%s\n' "$ENTRIES"   | grep    '^\[NEW\] \S' || true)"

STALE_COUNT="$([ -z "$STALE_LINES" ] && echo 0 || printf '%s\n' "$STALE_LINES" | wc -l | tr -d ' ')"
NEW_COUNT="$(  [ -z "$NEW_LINES"   ] && echo 0 || printf '%s\n' "$NEW_LINES"   | wc -l | tr -d ' ')"

echo ""
echo "✗ mem-cli: commit blocked — agent memory is stale"
echo ""

if [ "$STALE_COUNT" -gt 0 ]; then
  echo "Docs to regenerate ($STALE_COUNT):"
  printf '%s\n' "$STALE_LINES" | sed 's/^/  - /'
  echo ""
fi

if [ "$NEW_COUNT" -gt 0 ]; then
  echo "New source paths without a doc ($NEW_COUNT):"
  printf '%s\n' "$NEW_LINES" | sed 's/^\[NEW\] /  - /'
  echo ""
fi

echo "→ Open Claude Code in this repo and run:  /memcli-update"
echo ""
exit 1
