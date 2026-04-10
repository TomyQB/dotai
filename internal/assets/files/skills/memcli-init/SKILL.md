---
name: memcli-init
description: Bootstrap the .agent-memory/ directory for the current project, install the git pre-commit hook, and register the project in the mem-cli registry. Use when the user runs /memcli-init, asks to initialize mem-cli in a project, or wants to set up agent documentation for a new project.
---

# memcli-init — Bootstrap agent memory for a project

You are initializing mem-cli for the project in the current working directory. Follow these steps EXACTLY and in order.

## Step 1 — Verify context

1. Run `pwd` and `git rev-parse --show-toplevel` to confirm you are inside a git repository. If not, STOP and tell the user that mem-cli requires a git repo.
2. Check if `.agent-memory/` already exists. If yes, ask the user if they want to abort, or overwrite only missing files (do NOT destroy existing docs).

## Step 2 — Create `.agent-memory/` structure

Create the following files and directories at the repo root:

```
.agent-memory/
├── index.md
├── .stale                (empty file)
├── .gitignore            (empty, placeholder)
├── flows/.gitkeep
├── architecture/.gitkeep
├── api/.gitkeep
├── domain/.gitkeep
└── features/.gitkeep
```

Use the template below for `index.md`. Replace `{{PROJECT_NAME}}` with the repo folder name.

### Template: `.agent-memory/index.md`

```markdown
---
id: index
project: {{PROJECT_NAME}}
---

# Agent Memory — {{PROJECT_NAME}}

This directory is the **living documentation for AI agents**. It is not intended for humans.
Do not read it top-to-bottom. Search on-demand when you need context about flows, architecture, APIs, or domain.

## How agents use this

- **Flows** (`flows/`) — step-by-step user flows with Mermaid diagrams. Each file has BOTH functional context (what the user experiences) AND technical context (front↔back calls, side effects).
- **Architecture** (`architecture/`) — C4 diagrams, ADRs, cross-cutting decisions.
- **API** (`api/`) — contracts between frontend and backend per feature.
- **Domain** (`domain/`) — data model, ERD, ubiquitous language.
- **Features** (`features/`) — per-feature state and file map.

## Frontmatter contract

Every `.md` in this tree MUST have frontmatter declaring which source paths it observes:

\`\`\`yaml
---
id: unique-id
watches:
  - src/features/signup/**
  - src/api/signup/**
last_updated: YYYY-MM-DD
---
\`\`\`

The `Stop` hook uses `watches` to mark docs stale when relevant source files change.
`.stale` is the source of truth for "what needs updating". Pre-commit blocks if it is non-empty.

## Index
<!-- keep this list updated when adding new docs -->
- (empty — add entries here)
```

## Step 3 — Install the git pre-commit hook

The pre-commit hook template lives at `~/.claude/mem-cli/hooks/pre-commit.sh`.
Copy it to `.git/hooks/pre-commit` and `chmod +x` it. If a pre-commit hook already exists, do NOT overwrite — instead, append a call to the mem-cli hook and warn the user.

```bash
cp ~/.claude/mem-cli/hooks/pre-commit.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## Step 4 — Register in the mem-cli registry (v2 schema)

The registry lives at `~/.claude/memcli/projects.json` and uses schema version 2:

```json
{
  "version": 2,
  "projects": [
    {
      "path": "/absolute/path/to/repo",
      "name": "repo",
      "initialized_at": "2026-04-09T12:34:56Z",
      "last_seen": "2026-04-09T12:34:56Z"
    }
  ]
}
```

Flow:
1. Read `~/.claude/memcli/projects.json`. If it does not exist, treat as empty.
2. If the file is in **v1 shape** (`{"projects": ["/path", ...]}` — array of bare strings), migrate it in memory: for each string entry, produce an object with `path` = the string, `name` = `basename(path)`, `initialized_at` and `last_seen` = empty string or omit (the Go code accepts zero timestamps).
3. Upsert the current repo:
   - If an entry with the same `path` already exists, update its `last_seen` to the current time (RFC3339/ISO-8601 UTC, e.g. `2026-04-09T12:34:56Z`). **Do NOT overwrite `initialized_at`.**
   - Otherwise append a new entry with `path` (absolute repo path), `name` (basename), `initialized_at` = now, `last_seen` = now.
4. Write the file back in v2 shape (always include `"version": 2`). Use pretty-printed JSON with 2-space indent.

Create the parent directory (`~/.claude/memcli/`) if it does not exist. Never write the v1 shape.

## Step 5 — Confirm to the user

Print a summary:
- Created `.agent-memory/` with N files
- Installed git pre-commit hook
- Registered project in mem-cli registry
- Next step: optionally run `/memcli-scan` to generate initial docs from existing code, or start adding docs manually as you work.

## Rules

- Do NOT create fake/placeholder flows, APIs or architecture docs. The directories start empty and fill up as real work happens.
- Do NOT modify existing docs.
- Do NOT touch Engram — mem-cli and Engram are separate systems.
