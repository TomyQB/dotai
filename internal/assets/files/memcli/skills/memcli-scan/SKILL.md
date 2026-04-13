---
name: memcli-scan
description: Perform an initial full scan of an existing project to bootstrap .agent-memory/ with flows, architecture, API and domain docs by dispatching multiple explorer agents in parallel. Use ONLY when the user explicitly runs /memcli-scan or asks for an initial scan of an existing codebase.
---

# memcli-scan — Initial bootstrap scan (opt-in)

This is a HEAVY operation. Only run it when the user explicitly asks.

## Step 1 — Confirm

Ask the user to confirm: "This will dispatch multiple explorer agents to read the codebase and generate initial docs under `.agent-memory/`. It may take several minutes. Continue? (yes/no)"

If they say no, STOP.

## Step 2 — Verify `.agent-memory/` exists

If it does not, run `/memcli-init` first.

## Step 3 — Dispatch explorers in parallel

Launch the following agents in parallel (single message, multiple Agent tool calls):

1. **flows explorer** — identify top-level user flows (auth, signup, checkout, main features). For each, create `.agent-memory/flows/<flow>.md` with functional description + Mermaid sequence diagram + technical calls front↔back + frontmatter with `watches` pointing to the relevant source paths.

2. **architecture explorer** — detect overall architecture (monolith, hexagonal, layered, microservices). Create `.agent-memory/architecture/overview.md` with a Mermaid C4-style container diagram + narrative + frontmatter watching the top-level source dirs.

3. **api explorer** — inventory all HTTP endpoints (routes, controllers). Group by feature and create `.agent-memory/api/<feature>.md` with a table of endpoints, request/response shapes, and a Mermaid sequence for non-trivial ones.

4. **domain explorer** — identify core entities and relationships. Create `.agent-memory/domain/model.md` with a Mermaid ER diagram and a glossary of domain terms.

Each explorer MUST write frontmatter with accurate `watches` globs so future Stop hooks can mark things stale correctly.

## Step 4 — Update `.agent-memory/index.md`

Add the generated files to the Index section.

## Step 5 — Report

Tell the user: N flows, N arch docs, N API docs, N domain docs generated. Recommend they skim and correct anything off before committing.

## Rules

- Explorers MUST produce Mermaid diagrams (not ASCII, not prose-only).
- Every doc MUST have valid frontmatter with `watches`.
- Do NOT invent features or endpoints that are not in the code.
