# mem-cli

Living agent-facing documentation for your projects, enforced by hooks.

mem-cli installs a set of Claude Code skills, an agent (`doc-keeper`), and two
hooks that keep `.agent-memory/` in sync with your source code **automatically
and obligatorily**.

This documentation is **not for humans**. It is for AI agents that need
on-demand context about flows, APIs, architecture and domain when they work on
your codebase.

## Install

```bash
go build -o ~/.local/bin/memcli ./cmd/memcli
memcli install
```

This writes:

- `~/.claude/skills/memcli-{init,update,scan,doctor}/SKILL.md`
- `~/.claude/agents/doc-keeper.md`
- `~/.claude/mem-cli/hooks/{stop-hook.sh,pre-commit.sh}`
- Merges a `Stop` hook entry into `~/.claude/settings.json`

## Bootstrap a project

Open Claude Code inside any git repo and run:

```
/memcli-init
```

This creates `.agent-memory/`, installs the git pre-commit hook in
`.git/hooks/pre-commit`, and registers the project in
`~/.claude/memcli/projects.json`.

Optionally, for an existing codebase:

```
/memcli-scan
```

to dispatch explorer agents that generate initial flows, architecture, API and
domain docs from the current code.

## Workflow

1. You work normally with Claude Code on your repo.
2. When the agent stops, the `Stop` hook inspects the working tree, filters
   out changes that don't matter (tests, styles, types, renames, docs,
   lockfiles, assets) and cross-references the surviving paths against the
   `watches` frontmatter of every doc under `.agent-memory/`.
3. Any doc whose watches match a changed path is appended to
   `.agent-memory/.stale`.
4. When you try to `git commit`, the pre-commit hook reads `.stale`. If it is
   non-empty, the commit is **blocked** with a message telling you to run
   `/memcli-update` inside Claude Code.
5. `/memcli-update` reads `.stale` and dispatches the `doc-keeper` agent
   (one per stale doc, in parallel) to **regenerate each doc from scratch**
   based on the current source code. Only when all succeed, `.stale` is
   cleared. Then the commit goes through.

## Doc contract

Every `.md` under `.agent-memory/` must have frontmatter with a `watches` list:

```yaml
---
id: signup-flow
watches:
  - src/features/signup/**
  - src/api/signup/**
last_updated: 2026-04-09
---
```

Each doc must contain BOTH:

- **Functional context** — what the user experiences, with a Mermaid
  `flowchart` or `stateDiagram`.
- **Technical context** — front↔back calls in order, with a Mermaid
  `sequenceDiagram`, plus relevant files and side effects.

## Commands

- `memcli` — launch the interactive TUI (when stdout is a TTY)
- `memcli install` — install skills/agents/hooks globally
- `memcli doctor` — verify global + project state
- `memcli registry prune` — remove registry entries whose path no longer exists
- `memcli version` — print version

When stdout is **not** a TTY (piped, redirected, CI), `memcli` with no args
prints usage and exits 0 — it never starts the Bubbletea event loop and never
emits ANSI escape sequences.

Project-level operations (`init`, `update`, `scan`, `doctor`) live as Claude
Code skills and are invoked with `/memcli-<name>` from inside Claude Code, not
from the terminal.

## Interactive TUI

Running `memcli` with no arguments in a terminal opens a Bubbletea UI with
three screens:

- **Home** — list of registered projects with status badges (`OK`, `STALE`,
  `MISSING`). Press `enter` to open a project, `r` to refresh the registry
  from disk, `d` to run doctor on the highlighted project (shown in a modal),
  `q`/`esc` to quit.
- **Project** — tree of `<project>/.agent-memory/`. Rows referenced in
  `.stale` are marked. Press `enter` on a `.md` to preview, `esc` to go back.
- **Preview** — scrollable Glamour-rendered markdown. Standard viewport
  keys (`up`/`down`/`pgup`/`pgdn`/`j`/`k`), `esc` to go back.

The registry file lives at `~/.claude/memcli/projects.json` (schema v2, with
transparent read-migration from the legacy v1 string-array shape).

## Engram

mem-cli is **NOT** Engram. Engram is session/decision memory; mem-cli is
structural documentation of flows and architecture. They are kept strictly
separate.

## Status

v0.1 — CLI (`install`, `doctor`, `registry prune`), Bubbletea TUI project
browser, skills, agent, and hooks.
