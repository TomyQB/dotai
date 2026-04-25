---
name: full-review
description: >
  Run a complete quality gate on the currently changed files in the repo:
  discover and execute in parallel all available `*-audit` skills (security)
  and `*-review` skills (code quality / style / best practices), then always
  run the native `simplify` skill at the end. Use this skill whenever the user
  asks to review changes before committing, asks for a "full review", a
  "quality gate", or any variation like "audit y review antes de commitear",
  "revisá esto antes del commit", "corré un review completo", "pasá el quality
  gate". This is a standalone skill — it is NOT invoked automatically by any
  other skill; the user must run it explicitly when they want the gate. It
  does NOT commit, push, or open PRs — it only reviews.
---

# Full Review (Audit + Review + Simplify)

Orchestrate the repo's quality gates on the currently changed files: discover installed `*-audit` skills (security) and `*-review` skills (code quality), run them **in parallel as sub-agents**, then run the native `simplify` skill to review for reuse / quality / efficiency.

## Why this matters

Audit and review skills each have a narrow domain (`owasp-audit` for security, `java-review` for Java/Spring quality, `vue-review` for Vue, etc.). Running them one by one is slow and clutters the main conversation context. Running them as parallel sub-agents keeps the main thread clean and returns findings in a single consolidated report.

`simplify` runs after audits/reviews because simplification can change the code — if it ran first, audit/review output would point at lines that no longer exist.

## Flow

### Step 1: Collect the changed files

Run these in parallel and merge the results into a single file list:

```bash
git diff --staged --name-only
git diff --name-only
git ls-files --others --exclude-standard
```

If the merged list is empty, stop — there's nothing to review.

### Step 2: Skip if there's no source code

Classify each file in the list:

| File extension / path | Category |
|---|---|
| `.md`, `.txt` | documentation |
| `.json`, `.yml`, `.yaml`, `.toml` | config / CI |
| Anything else | source code |

**If the changeset contains ONLY documentation or config files** → skip the entire flow. Tell the user briefly ("solo docs/config — no hay nada que auditar/revisar") and exit.

If there is **at least one source file**, continue. Audit/review sub-agents will operate on the full changed-file list (they filter internally by their own scope — `java-review` looks at `.java`, `vue-review` at `.vue`, etc.).

### Step 3: Discover `*-audit` and `*-review` skills

**Local first** — search the project's `.claude/skills/`:

```bash
ls -d .claude/skills/*-audit .claude/skills/*-review 2>/dev/null \
  | while read d; do [ -f "$d/SKILL.md" ] && echo "$d"; done
```

**Global fallback** — only if the local search returned NOTHING, search `~/.claude/skills/`:

```bash
ls -d ~/.claude/skills/*-audit ~/.claude/skills/*-review 2>/dev/null \
  | while read d; do [ -f "$d/SKILL.md" ] && echo "$d"; done
```

**Rule:** all skills matched at the SAME level run in parallel. Do not mix local + global results — local wins whole-or-nothing. This lets a project override global skills (e.g., a project-specific `java-review` in `.claude/skills/` takes precedence over the global one).

### Step 4: Run audit + review skills in parallel (sub-agents)

For each discovered skill, launch a **parallel sub-agent** via the Agent tool (`subagent_type: general-purpose`) with:

- The skill's `SKILL.md` path
- The full list of changed files (from Step 1)
- A clear instruction: "Read the skill's SKILL.md and apply its review rules ONLY to the changed files listed. Do NOT scan the entire codebase. Return a structured report with findings grouped by severity (Critical / Must Fix / Warning / Info)."

**Send all sub-agents in a single message with multiple Agent tool calls** so they run concurrently. Wait for all of them to finish before continuing.

### Step 5: Run `simplify` (always)

After all audit/review sub-agents return (or immediately if none were discovered), invoke the native `simplify` skill via the Skill tool:

```
Skill(skill: "simplify")
```

`simplify` may modify files in place. If it does, **tell the user to re-stage those files** before proceeding to the next step (commit / PR / whatever is next). Do NOT re-stage automatically — the user decides whether the simplification is acceptable.

### Step 6: Consolidate and report

Build a single report for the user, grouping findings from all audit/review sub-agents plus the `simplify` result. Group by severity:

- **🔴 Critical / Must Fix** — highlight prominently. Ask the user whether to stop and fix before continuing, or proceed anyway. Never auto-fix critical findings.
- **🟡 Warning** — show as a list; continue by default.
- **🔵 Info / Nice to Have** — one-line summary; continue silently.
- **✅ No issues** — single line "Quality gate passed", nothing else.

If `simplify` changed files, append a final note:
```
⚠️  simplify modified N files. Re-stage with: git add <files> before next step.
```

## Edge cases

- **No audit/review skills installed anywhere**: skip Steps 3-4 and go straight to Step 5 (`simplify` always runs).
- **A sub-agent times out or errors**: report the failure for that specific skill but continue with the rest. Do not abort the whole review because one sub-agent failed.
- **Sub-agent returns free-form text instead of a structured report**: accept it, quote the raw output under that skill's section, and note the format deviation. Do not reject.
- **`simplify` fails or is not available**: skip it and note "simplify unavailable" in the final report. The audit/review results are still valid on their own.
- **Changeset is HUGE** (hundreds of changed files): warn the user before launching sub-agents ("voy a lanzar N sub-agents sobre M archivos, ¿continúo?"). Lets them narrow scope or split the review into multiple passes.
- **File list includes deleted files**: filter them out before sending to sub-agents. Audit/review on a deleted file makes no sense.
- **Called from another skill (e.g., `commit-and-push`)**: same behavior, but the "ask user whether to fix" decision for Critical findings should be forwarded to the caller's flow, not short-circuit the parent skill silently.
