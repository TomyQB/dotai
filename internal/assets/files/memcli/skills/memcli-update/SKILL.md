---
name: memcli-update
description: Update all stale documentation in .agent-memory/ by delegating to doc-keeper (for existing docs) or to explorer agents (for orphaned paths). Use when the user runs /memcli-update, when a pre-commit hook has blocked a commit due to stale docs, or when the user asks to refresh agent memory.
---

# memcli-update — Refresh stale agent memory

You are updating documentation that the Stop hook has marked as stale.
`.stale` contains two kinds of entries:

- `path/to/doc.md` — an existing doc whose source files changed. Delegate to
  `doc-keeper` to regenerate it from scratch.
- `[NEW] path/to/source/file.ext` — a changed source file that no existing
  doc's `watches` covers. Dispatch an explorer to **create a new doc** for it.

## Step 1 — Read `.agent-memory/.stale`

Read `.agent-memory/.stale`. If it is empty or does not exist, tell the user
"Nothing to update" and STOP.

Split the lines into two buckets:

| Bucket | Match | Action |
|--------|-------|--------|
| STALE  | does NOT start with `[NEW] `               | regenerate via `doc-keeper` |
| NEW    | starts with `[NEW] ` (strip the prefix)    | create new doc via explorer |

## Step 2 — Regenerate stale docs (existing)

For each STALE entry, launch the `doc-keeper` agent in parallel with the doc
path as input. `doc-keeper` reads the doc's `watches`, reads source, and
rewrites the doc preserving `id` and updating `last_updated`.

## Step 3 — Create new docs (orphans)

Group NEW entries by **suggested doc type** using these path heuristics.
Rules are **first-match-wins** top-to-bottom — a path like
`src/api/user.service.ts` matches the `flow` row only if it lives under
`views/`/`pages/`/`screens/` first; otherwise it falls through to `api` via
`**/api/**`. Stop at the first matching row.

| # | Path pattern                                          | Doc type     | Target folder                 |
|---|-------------------------------------------------------|--------------|-------------------------------|
| 1 | `**/views/**`, `**/pages/**`, `**/screens/**`         | flow         | `.agent-memory/flows/`        |
| 2 | `**/*.service.*`, `**/api/**`, `**/controllers/**`, `**/routes/**`, `**/endpoints/**` | api  | `.agent-memory/api/`          |
| 3 | `**/types/**`, `**/models/**`, `**/schemas/**`, `**/entities/**`                       | domain | `.agent-memory/domain/`       |
| 4 | `**/composables/**`, `**/hooks/**`, `**/stores/**`                                     | feature | `.agent-memory/features/`    |
| 5 | anything else                                                                          | architecture | `.agent-memory/architecture/` |

**Group by directory root** so paths that are clearly part of the same module
produce a single doc. Example: `src/views/reception/guests/CreateView.vue`,
`src/views/reception/guests/ActiveView.vue`, `src/views/reception/guests/HistoryView.vue`
→ one flow doc (not three).

For each group, dispatch a sub-agent (general-purpose or Explore) with this
prompt template:

```
You are creating a new .agent-memory/{type}/{slug}.md doc for this project.

Read these source paths and write the doc:
- {path1}
- {path2}
- ...

Follow the exact format used by the doc-keeper agent:
- Frontmatter with: `id: {slug}`, `watches: [<glob patterns covering the paths above>]`, `last_updated: {today}`.
- "Functional context" section (what the user experiences, plain language) +
  a Mermaid `flowchart` or `stateDiagram`.
- "Technical context" section (what happens in code) + a Mermaid
  `sequenceDiagram` showing calls in order.
- "Relevant files" list and "Side effects" list.

Watches rules:
- Use recursive globs like `src/views/reception/guests/**` when the files live
  in a folder together.
- Include every path that materially belongs to this doc (services,
  composables, types used by those components).
- Never fabricate — only reference code that exists.
```

Pick `{slug}` from the folder/feature name (kebab-case, no extension).
Example: `src/views/reception/guests/**` → `flows/reception-guest-reservations.md`.

After the sub-agent writes the file, append the new doc to `.agent-memory/index.md`
under the correct section (Flows / Architecture / API / Domain / Features).

## Step 4 — Clear `.stale`

Only after ALL delegations (doc-keeper and explorers) return successfully,
truncate `.agent-memory/.stale` to empty.
If any delegation failed, keep the failed entries in `.stale` and report which
ones failed.

## Step 5 — Report

Summarize for the user:
- N existing docs regenerated (list paths).
- M new docs created (list type + path + source paths).
- K failures (if any, with reason).
- "You can commit now" when `.stale` is empty.

## Rules

- NEVER clear `.stale` if any update failed.
- NEVER modify existing docs directly — always via `doc-keeper`.
- NEVER fabricate endpoints, flows, or entities. Only describe code that exists.
- Preserve frontmatter `id` and `watches` when regenerating. Update `last_updated` to today.
- When creating a new doc, pick `watches` globs wide enough to cover the
  module but not the whole repo. Prefer the folder containing the files.
- Run delegations in parallel when they are independent.
