#!/bin/bash
input=$(cat)

# Extract values from JSON
model=$(echo "$input" | jq -r '.model.display_name // "Unknown"')
dir=$(echo "$input" | jq -r '.workspace.current_dir // "."')
project=$(basename "$dir")
branch=$(git -C "$dir" --no-optional-locks symbolic-ref --short HEAD 2>/dev/null)
ctx_pct=$(echo "$input" | jq -r '.context_window.used_percentage // 0')
style=$(echo "$input" | jq -r '.output_style.name // "default"')
duration_ms=$(echo "$input" | jq -r '.cost.total_duration_ms // 0')

# Colors
CYAN='\033[36m'
GREEN='\033[32m'
RESET='\033[0m'

# Line 1: [Model] 📁 project | 🌿 branch
line1="${CYAN}${model}${RESET} | 📁 ${project}"
if [ -n "$branch" ]; then
  line1="${line1} | 🌿 ${branch}"
else
  line1="${line1} | no-git"
fi

# Progress bar (10 blocks)
ctx_int=$(printf '%.0f' "$ctx_pct" 2>/dev/null || echo 0)
[ "$ctx_int" -gt 100 ] 2>/dev/null && ctx_int=100
[ "$ctx_int" -lt 0 ] 2>/dev/null && ctx_int=0
filled=$((ctx_int / 10))
empty=$((10 - filled))
bar=""
for ((i=0; i<filled; i++)); do bar="${bar}█"; done
for ((i=0; i<empty; i++)); do bar="${bar}░"; done

# Duration (ms -> Xm Ys)
duration_sec=$(( ${duration_ms%.*} / 1000 ))
minutes=$((duration_sec / 60))
seconds=$((duration_sec % 60))

# Line 2: progress bar % | style | ⏱ time
line2="${GREEN}${bar}${RESET} ${ctx_int}% | ${GREEN}${style}${RESET} | ⏱ ${minutes}m ${seconds}s"

echo -e "${line1}\n${line2}"
