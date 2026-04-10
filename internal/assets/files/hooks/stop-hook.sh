#!/usr/bin/env bash
# mem-cli Stop hook
# Runs at the end of each Claude Code turn.
# Detects source files that were modified in the working tree, filters out
# changes that "do not matter" (tests, styles, types, renames), and cross-
# references the surviving paths against the `watches` frontmatter of every
# .md under .agent-memory/. Any doc whose watches match a changed path is
# appended to .agent-memory/.stale (deduplicated).
#
# Exits 0 always — this hook must NEVER block the agent's turn. Blocking
# happens later in the git pre-commit hook.

set -u

# Read the hook JSON payload from stdin but we don't need it — we use git.
cat >/dev/null 2>&1 || true

# Find the project root (must be a git repo). If not, silently exit.
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$REPO_ROOT" || exit 0

MEM_DIR=".agent-memory"
[ -d "$MEM_DIR" ] || exit 0

STALE_FILE="$MEM_DIR/.stale"
touch "$STALE_FILE"

# --- Collect changed files in the working tree (staged + unstaged, excluding deleted)
CHANGED="$(git status --porcelain 2>/dev/null | awk '
  {
    status=substr($0,1,2)
    path=substr($0,4)
    # skip pure deletions
    if (status ~ /D/) next
    # handle rename "old -> new"
    if (index(path, " -> ")) {
      n=index(path, " -> ")
      path=substr(path, n+4)
    }
    print path
  }
')"

[ -z "$CHANGED" ] && exit 0

# --- Static filter: "changes that do not matter"
# Skip tests, styles, assets, pure type declaration files, lockfiles, docs,
# and anything already under .agent-memory itself.
FILTERED=""
while IFS= read -r p; do
  [ -z "$p" ] && continue
  case "$p" in
    .agent-memory/*) continue ;;
    .git/*) continue ;;
    *.md|*.mdx|*.txt|*.rst) continue ;;
    *.test.*|*.spec.*|*_test.go) continue ;;
    */__tests__/*|*/tests/*|*/test/*) continue ;;
    *.css|*.scss|*.sass|*.less|*.styl) continue ;;
    *.svg|*.png|*.jpg|*.jpeg|*.gif|*.webp|*.ico) continue ;;
    *.d.ts) continue ;;
    */types/*|*/typings/*) continue ;;
    *.lock|*-lock.json|*.sum|go.sum) continue ;;
  esac
  FILTERED="$FILTERED
$p"
done <<EOF
$CHANGED
EOF

FILTERED="$(printf '%s\n' "$FILTERED" | sed '/^$/d')"
[ -z "$FILTERED" ] && exit 0

# --- For each .md under .agent-memory, read its `watches` frontmatter and
#     check if any changed (filtered) file matches any watch glob.
NEW_STALE=""
while IFS= read -r -d '' doc; do
  # extract lines between first two '---' markers
  fm="$(awk 'BEGIN{n=0} /^---[[:space:]]*$/{n++; next} n==1{print} n>=2{exit}' "$doc")"
  [ -z "$fm" ] && continue

  # parse watches: lines that start with "  - "
  watches="$(printf '%s\n' "$fm" | awk '
    /^watches:[[:space:]]*$/{inw=1; next}
    inw==1 && /^[[:space:]]*-[[:space:]]*/ { sub(/^[[:space:]]*-[[:space:]]*/, ""); print; next }
    inw==1 && /^[^[:space:]]/ { inw=0 }
  ')"
  [ -z "$watches" ] && continue

  rel_doc="${doc#./}"

  matched=0
  while IFS= read -r glob; do
    [ -z "$glob" ] && continue
    # strip surrounding quotes if any
    glob="${glob%\"}"; glob="${glob#\"}"
    glob="${glob%\'}"; glob="${glob#\'}"
    while IFS= read -r f; do
      [ -z "$f" ] && continue
      case "$f" in
        $glob) matched=1; break 2 ;;
      esac
    done <<FEOF
$FILTERED
FEOF
  done <<GEOF
$watches
GEOF

  if [ "$matched" = "1" ]; then
    NEW_STALE="$NEW_STALE
$rel_doc"
  fi
done < <(find "$MEM_DIR" -type f -name '*.md' -print0)

[ -z "$NEW_STALE" ] && exit 0

# --- Merge into .stale (dedup, stable order)
{
  cat "$STALE_FILE"
  printf '%s\n' "$NEW_STALE"
} | sed '/^$/d' | awk '!seen[$0]++' > "$STALE_FILE.tmp"
mv "$STALE_FILE.tmp" "$STALE_FILE"

exit 0
