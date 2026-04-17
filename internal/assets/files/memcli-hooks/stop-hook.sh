#!/usr/bin/env bash
# mem-cli Stop hook — runs at the end of each Claude Code turn.
#
# For every file changed in the working tree:
#   - STALE: if any .agent-memory/*.md doc's `watches` covers it, append that
#     doc's path to .agent-memory/.stale so /memcli-update regenerates it.
#   - NEW:   if no existing doc covers it, append "[NEW] <path>" to .stale so
#     /memcli-update dispatches an explorer to create a brand-new doc.
#
# Exits 0 always — never blocks the turn. Blocking happens in the git
# pre-commit hook when .stale is non-empty.

set -u

# Drain any hook JSON payload from stdin (we rely on git, not the payload).
cat >/dev/null 2>&1 || true

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$REPO_ROOT" || exit 0

[ -d ".agent-memory" ] || exit 0

MEM_DIR=".agent-memory"
STALE_FILE="$MEM_DIR/.stale"
touch "$STALE_FILE"

# Graceful no-op if the host lacks python3. The hook's contract is "never
# block the turn", and spewing a traceback every time would violate it.
command -v python3 >/dev/null 2>&1 || exit 0

CHANGED="$(git status --porcelain 2>/dev/null | awk '
  {
    status=substr($0,1,2)
    path=substr($0,4)
    if (status ~ /D/) next
    if (index(path, " -> ")) {
      n=index(path, " -> ")
      path=substr(path, n+4)
    }
    print path
  }
')"

[ -z "$CHANGED" ] && exit 0

# Static filter — ignore noise categories (tests, styles, assets, type-only
# files, lockfiles, docs, and anything under .agent-memory/ itself). Kept in
# lockstep with the list documented in memcli/agents/doc-keeper.md.
FILTERED=""
while IFS= read -r p; do
  [ -z "$p" ] && continue
  case "$p" in
    .agent-memory/*|*.agent-memory/*) continue ;;
    *.test.*|*.spec.*|*_test.go|*_spec.rb|test/*|tests/*|__tests__/*) continue ;;
    *.css|*.scss|*.sass|*.less) continue ;;
    *.svg|*.png|*.jpg|*.jpeg|*.gif|*.ico|*.webp) continue ;;
    *.d.ts) continue ;;
    package-lock.json|yarn.lock|pnpm-lock.yaml|composer.lock|Gemfile.lock|go.sum|Cargo.lock) continue ;;
    *.md|*.markdown) continue ;;
    *) ;;
  esac
  FILTERED="$FILTERED
$p"
done <<EOF
$CHANGED
EOF

FILTERED="$(printf '%s\n' "$FILTERED" | sed '/^$/d')"
[ -z "$FILTERED" ] && exit 0

# Delegate glob-matching to the Python helper that ships next to this hook
# (bash `case $glob` cannot express `**` recursion).
MATCHER="$(dirname "$0")/stop-hook-matcher.py"
[ -f "$MATCHER" ] || exit 0

CLASSIFICATION="$(printf '%s\n' "$FILTERED" | python3 "$MATCHER" "$MEM_DIR")"
[ -z "$CLASSIFICATION" ] && exit 0

NEW_STALE="$(printf '%s\n' "$CLASSIFICATION" | awk '
  $1 == "STALE" { sub(/^STALE /, ""); print; next }
  $1 == "NEW"   { sub(/^NEW /, "");   print "[NEW] " $0; next }
')"

[ -z "$NEW_STALE" ] && exit 0

{
  cat "$STALE_FILE"
  printf '%s\n' "$NEW_STALE"
} | awk 'NF && !seen[$0]++' > "$STALE_FILE.tmp"
mv "$STALE_FILE.tmp" "$STALE_FILE"

exit 0
