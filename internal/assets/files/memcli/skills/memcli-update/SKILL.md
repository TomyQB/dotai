---
name: memcli-update
description: Update all stale documentation in .agent-memory/ by delegating to the doc-keeper agent. Use when the user runs /memcli-update, when a pre-commit hook has blocked a commit due to stale docs, or when the user asks to refresh agent memory.
---

# memcli-update — Refresh stale agent memory

You are updating documentation that the Stop hook has marked as stale.

## Step 1 — Read `.agent-memory/.stale`

Read the file at `.agent-memory/.stale` in the repo root. Each non-empty line is a path (relative to repo root) of a `.md` file that needs regeneration.

If the file is empty or does not exist, tell the user "Nothing to update" and STOP.

## Step 2 — For each stale doc, delegate to `doc-keeper`

For each stale doc path, launch the `doc-keeper` agent with the doc path as input. Run delegations in parallel when there is no dependency between docs.

The `doc-keeper` will:
1. Read the doc's frontmatter to discover `watches` paths.
2. Read the current source code under those paths.
3. **Regenerate the doc from scratch** (functional + technical Mermaid) preserving the frontmatter `id`, updating `last_updated`.

## Step 3 — Clear `.stale`

Only after ALL doc-keeper delegations return successfully, truncate `.agent-memory/.stale` to empty.
If any delegation failed, keep the failed entries in `.stale` and report which ones failed.

## Step 4 — Report

Summarize for the user: which docs were regenerated, which failed (if any), and that they can now commit.

## Rules

- NEVER clear `.stale` if any update failed.
- NEVER modify docs directly — always via `doc-keeper`.
- Preserve frontmatter `id` and `watches`. Update `last_updated` to today.
