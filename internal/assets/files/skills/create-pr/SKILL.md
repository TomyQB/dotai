---
name: create-pr
description: >
  Create (or refresh) a GitHub Pull Request for the current branch, with a
  description built from FUNCTIONAL context only — never technical implementation
  detail. Reviewers read the description to understand what was asked, then look
  at the diff to validate that the technical changes meet those requirements.
  Use this skill whenever the user asks to open, create, or refresh a PR — any
  variation like "create pr", "open pr", "abrí un PR", "abrí pr", "crear pr",
  "actualizá el PR", "refresh the pr", "update pr description". Also trigger
  after a successful commit-and-push when the user says "y ahora abrí el PR".
  This skill does NOT commit or push — use the separate `commit-and-push` skill
  for that. It assumes the branch is already pushed to origin.
---

# Create PR with Functional Context

Automate the creation (or refresh) of a Pull Request on GitHub. The PR body contains **only functional context** — what was asked and the expected behavior. Reviewers validate that the technical diff implements those requirements; they do NOT need the description to repeat what the code already says.

## Why this matters

A good PR description answers **what** was asked and **why**. The code itself answers **how**. Mixing both makes reviews lazy: reviewers end up reading the description and nodding along instead of verifying the diff actually implements the contract.

By keeping the description strictly functional, every reviewer has the same mental model going in and must use the diff to judge implementation quality.

## Pre-conditions

This skill assumes:
- The user has committed and pushed their changes (via `commit-and-push` or manually)
- The current branch has an upstream tracking a remote branch
- `gh` CLI is authenticated and has access to the repo

If any of these are missing, the skill reports the gap and stops — it does not commit, stage, or push.

## Flow

### Step 1: Verify pre-conditions

Run these in parallel:

```bash
git branch --show-current              # the source branch
git rev-parse --abbrev-ref @{upstream} # the remote tracking branch, if any
git status -sb                         # any uncommitted changes?
gh auth status                         # auth ok?
```

If any of the following is true, STOP and tell the user to resolve first:
- Current branch has no upstream → tell user to `git push -u origin <branch>` or run the `commit-and-push` skill
- There are unstaged or uncommitted changes → tell user those won't be in the PR; ask whether to continue anyway or stop and commit first
- `gh auth status` fails → tell user to authenticate with `gh auth login`

### Step 2: Determine the base branch (PR target)

Decide the target from the current branch prefix:

| Branch prefix | Default PR target |
|---|---|
| `feature/*` | `develop` |
| `fix/*` | `develop` |
| `hotfix/*` | `main` |
| `release/*` | `main` |
| Anything else | Ask the user via `AskUserQuestion` which branch to target |

Before proceeding, verify the target exists on the remote:

```bash
git ls-remote --heads origin <target>
```

If the target does NOT exist (e.g., repo without `develop`), tell the user clearly and ask for an alternative target via `AskUserQuestion`. Do NOT silently fall back.

Branch-pattern validation (i.e., enforcement that `feature/*` must target `develop`) is NOT this skill's responsibility — that's the repo's GitHub Actions workflow job. This skill opens the PR and lets the action flag any violation.

### Step 3: Resolve the functional context (strict priority order)

Use the FIRST source that yields usable content — do NOT fabricate or merge sources:

**Priority 1 — Engram / SDD spec lookup** (if `mem_search` is available):
- Derive search keywords from: the branch tail (e.g. `feature/oauth-login` → `oauth login`), and the subject of the latest commit
- Search engram with those keywords and also try these topic keys explicitly:
  - `sdd/<branch-tail>/spec`
  - `sdd/<branch-tail>/proposal`
- If a match is found, call `mem_get_observation` for the full content. Extract only functional requirements, scenarios, acceptance criteria. **Skip** technical design, architecture decisions, and implementation notes.

**Priority 2 — Current conversation context**:
- If the user stated what they wanted earlier in this session (e.g. "implementá login con OAuth", "arreglá el bug que tira 500 en checkout"), use that statement as the functional context.

**Priority 3 — Ask the user** via `AskUserQuestion`:
- Question: `"¿Cuál es el contexto funcional de este cambio? (qué se pidió y qué comportamiento se espera)"`
- Free-form answer.

If priorities 1 and 2 both fail and the user opts to skip question 3, fall back to a placeholder (see Edge cases).

### Step 4: Build title and body

**Title** — reuse the subject of the latest conventional commit on the branch:

```bash
git log -1 --pretty=format:"%s"
```

(e.g., `feat(auth): add OAuth login`.) If the branch has multiple commits, pick the most significant commit's subject.

**Body** — use this EXACT Markdown structure. Omit empty sections entirely rather than leaving them blank:

```markdown
## Contexto funcional

<párrafo corto: qué se pidió y para qué — NO detalles de implementación>

## Comportamiento esperado

- <criterio de aceptación 1>
- <criterio de aceptación 2>
- ...

## Fuera de alcance

- <qué NO resuelve este PR — solo si evita confusión>

---

## Para el reviewer

Revisá los diffs y verificá que la implementación técnica cumple los criterios de arriba. Los detalles de "cómo" están en el código; esta descripción cubre solo el "qué" y el "por qué".
```

**Golden rule for the body**: ZERO mentions of:
- File paths (`src/auth.ts`)
- Function / class / method names
- Libraries or frameworks
- Any implementation detail

Speak only in terms of **user-observable behavior** and **requirements**.

Write the body to a tempfile to avoid shell escaping issues:

```bash
cat > /tmp/pr-body.md <<'EOF'
<body here>
EOF
```

### Step 5: Check for an existing open PR from this branch

```bash
gh pr list --head "$(git branch --show-current)" --state open \
  --json number,url --jq '.[0]'
```

### Step 6: Create or refresh

**If NO open PR exists** → create one (no confirmation prompt; the repo's workflow validates branch naming):

```bash
gh pr create --base <target> --head "$(git branch --show-current)" \
  --title "<title>" --body-file /tmp/pr-body.md
```

**If an open PR already exists** → fully rewrite its body (title stays):

```bash
gh pr edit <pr-number> --body-file /tmp/pr-body.md
```

The rewrite is unconditional and intentional: re-running this skill always produces a body that reflects the current session's understanding of the requirement. Manual edits to the body made in GitHub UI WILL be lost.

### Step 7: Report

Show the user:
- The PR URL (newly created or refreshed)
- Whether it was created or the body was rewritten
- The base branch targeted
- A one-line warning if the body came from the placeholder (so they know to complete it before review)

## Edge cases

- **Branch has no upstream**: Stop and tell the user to push first. Do NOT silently push — that's the responsibility of `commit-and-push`.
- **Uncommitted changes in the working tree**: Warn the user and ask if they want to proceed anyway; those changes will NOT be in the PR.
- **Target branch missing in remote**: Ask the user for an alternative target via `AskUserQuestion`. Never fall back silently.
- **`gh pr create` fails** (auth, missing scope, no permission): Report the full gh CLI error verbatim. Common causes:
  - Token missing `repo` scope
  - Token missing `workflow` scope when the PR also touches workflow files (not this skill's concern, but the error will appear)
  - Active `gh` account has no access to the repo (collaborator not added)
  - PAT expired
  Do NOT retry blindly — surface the error and let the user decide.
- **`mem_search` unavailable** (engram MCP not installed or project not indexed): Skip priority 1 silently and fall through to priority 2.
- **Functional context unresolvable** (engram empty, no session context, user opts to skip the question): Use a placeholder body and mark the report so the user knows:
  ```markdown
  ## Contexto funcional

  _Pendiente: completar antes de review._

  ---

  ## Para el reviewer

  Revisá los diffs y verificá que la implementación técnica cumple lo que se pedirá arriba (actualmente pendiente).
  ```
- **Branch matches multiple patterns ambiguously**: Prioritize the more specific prefix (`hotfix/` over `fix/` if both could match). If still ambiguous, ask the user.
- **PR already exists on a DIFFERENT base branch** than the one this skill wants to target: Report the discrepancy and ask the user whether to rewrite the body anyway, change the base branch with `gh pr edit --base`, or stop.
- **User asks to run this skill on `main` / `develop` / protected branch directly**: No-op, tell the user there's nothing to PR — those branches receive PRs, they don't open them.
