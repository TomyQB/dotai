---
name: memcli-doctor
description: Verify that mem-cli is correctly installed and that the current project (if any) is properly initialized. Use when the user runs /memcli-doctor or asks to check mem-cli health.
---

# memcli-doctor — Verify installation and project state

## Step 1 — Global installation checks

Verify the following exist:
- `~/.claude/skills/memcli-init/SKILL.md`
- `~/.claude/skills/memcli-update/SKILL.md`
- `~/.claude/skills/memcli-scan/SKILL.md`
- `~/.claude/skills/memcli-doctor/SKILL.md`
- `~/.claude/agents/doc-keeper.md`
- `~/.claude/mem-cli/hooks/stop-hook.sh` (executable)
- `~/.claude/mem-cli/hooks/pre-commit.sh`
- `~/.claude/settings.json` contains a `Stop` hook pointing to `stop-hook.sh`

Report any missing items and tell the user to run `memcli install` from the terminal to fix.

## Step 2 — Project checks (if inside a git repo)

If cwd is a git repo:
- `.agent-memory/` exists
- `.agent-memory/index.md` exists and has valid frontmatter
- `.git/hooks/pre-commit` exists and references mem-cli
- The repo path is listed in `~/.config/memcli/registry.json`
- `.agent-memory/.stale` — report how many entries are currently stale

## Step 3 — Report

Print a concise status table: ✓ / ✗ per check. For any ✗, give the exact fix command.
