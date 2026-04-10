#!/usr/bin/env bash
# mem-cli git pre-commit hook
# Blocks the commit if .agent-memory/.stale is non-empty.
# Tells the user to run /memcli-update inside Claude Code.

set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
STALE_FILE="$REPO_ROOT/.agent-memory/.stale"

if [ ! -f "$STALE_FILE" ]; then
  exit 0
fi

# count non-empty lines
COUNT=$(sed '/^$/d' "$STALE_FILE" | wc -l | tr -d ' ')

if [ "$COUNT" = "0" ]; then
  exit 0
fi

echo ""
echo "✗ mem-cli: commit blocked — agent memory is stale"
echo ""
echo "The following docs must be regenerated before committing:"
echo ""
sed '/^$/d' "$STALE_FILE" | sed 's/^/  - /'
echo ""
echo "→ Open Claude Code in this repo and run:  /memcli-update"
echo ""
exit 1
