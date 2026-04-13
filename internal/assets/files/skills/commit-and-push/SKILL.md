---
name: commit-and-push
description: >
  Commit and push changes following the Conventional Commits specification.
  Use this skill whenever the user asks to commit, push, save changes to git,
  or any variation like "commit and push", "sube los cambios", "haz commit",
  "commitea", "push this", "save my work to git". Also trigger when the user
  finishes a task and says something like "done, push it" or "listo, sube eso".
  This skill handles the full flow: analyzing changes, generating the commit
  message, committing, verifying git account, and pushing.
---

# Conventional Commit & Push

Automate the full commit + push flow using the [Conventional Commits](https://www.conventionalcommits.org/) specification.

## Why this matters

Consistent commit messages make git history searchable, enable automated changelogs, and communicate intent clearly. This skill removes the friction of writing well-formatted messages by analyzing the actual changes and generating the right message.

## Flow

### Step 1: Analyze changes

Run these commands in parallel to understand the current state:

```bash
git status
git diff --staged
git diff
git log --oneline -5
```

If there are no changes at all (no staged, unstaged, or untracked files), stop and tell the user there's nothing to commit.

If there are unstaged or untracked files, determine which ones are relevant. Stage files that are clearly part of the work. If unsure which files to include, ask the user.

**Never stage files that likely contain secrets** (`.env`, `credentials.json`, `*.key`, etc.). Warn the user if these appear in the changes.

### Step 1.5: Project quality gates (audit, review & simplify)

**Skip this entire step when:**
- Only documentation files changed (`.md`, `.txt`, `.json`, `.yml`, `.yaml`, `.toml`)
- Only config/CI files changed (no source code in the diff)

**If source code files are in the changeset** (not just docs, config, or markdown):

**1. Discover audit & review skills:**

Search for skills whose directory name matches `*-audit` or `*-review` (each must contain a `SKILL.md`):

1. **Local first** — Search in the project's `.claude/skills/` directory
2. **Global fallback** — Only if NO matching skills were found locally, search in `~/.claude/skills/`

All matching skills within the same level run in parallel (e.g., `java-review` + `vue-review` both in local → both run).

```bash
# Local search
ls -d .claude/skills/*-audit .claude/skills/*-review 2>/dev/null | while read d; do [ -f "$d/SKILL.md" ] && echo "$d"; done

# Global search (only if no local results)
ls -d ~/.claude/skills/*-audit ~/.claude/skills/*-review 2>/dev/null | while read d; do [ -f "$d/SKILL.md" ] && echo "$d"; done
```

**2. Run audit & review skills (if any found):**
- Launch **all** discovered skills as **parallel subagents** (Agent tool) to keep the main context clean
- Each subagent should read the skill's `SKILL.md` and execute its instructions against the changed source files only (not the entire codebase)
- Wait for results

**After subagents complete:**
- If **Critical** or **Must Fix** issues are found: show them to the user and ask whether to fix before committing or proceed anyway
- If only **Low/Informational/Nice to Have** issues: show a brief summary and continue with the commit flow
- If no issues found: continue silently

**3. Simplify (always — native skill):**
After audit/review complete (or immediately if no audit/review skills were found), invoke the native `/simplify` skill:
- Use the `Skill` tool with `skill: "simplify"` to review changed code for reuse, quality, and efficiency
- If simplify makes changes to files, re-stage the modified files with `git add` before continuing to Step 2
- If simplify finds no issues: continue silently

### Step 2: Determine the commit type

Based on the diff content, classify the change into one of these types:

| Type | When to use |
|------|-------------|
| `feat` | A new feature or capability |
| `fix` | A bug fix |
| `docs` | Documentation only |
| `style` | Formatting, whitespace, semicolons — no logic change |
| `refactor` | Code restructuring without changing behavior |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `build` | Build system or dependencies (npm, cargo, pip, foundry) |
| `ci` | CI/CD configuration (GitHub Actions, etc.) |
| `chore` | Maintenance tasks, config updates, cleanup |
| `revert` | Reverting a previous commit |

If the changes span multiple types, use the **most significant** one. A feature that also updates tests is `feat`, not `test`.

### Step 3: Detect the scope

Look at which directories/modules the changes touch. If all changes are within a single module or directory, use it as scope:

- Changes in `src/auth/` → `feat(auth):`
- Changes in `contracts/` → `fix(contracts):`
- Changes in `test/` → `test(unit):`
- Changes spread across many areas → no scope, just `feat:`

Keep scopes short (1 word if possible). Don't force a scope if the changes are broad.

### Step 4: Detect breaking changes

If the diff shows:
- Removed or renamed public API functions/methods
- Changed function signatures (parameters added/removed/reordered)
- Deleted exported types/interfaces
- Changed database schemas or contract interfaces

Mark it as a breaking change with `!` after the scope:

```
feat(api)!: replace authentication endpoint
```

### Step 5: Generate the commit message

Format: `<type>(<scope>): <description>`

Rules for the description:
- **Lowercase** first letter, no period at the end
- **Imperative mood**: "add feature" not "added feature" or "adds feature"
- **Concise**: under 72 characters for the subject line
- Focus on **why** the change exists, not what files changed

For complex changes, add a body separated by a blank line explaining the reasoning.

**No trailers, no co-author lines, no attribution.** The commit message must contain ONLY the conventional commit message itself — nothing else. The commit must look entirely human-authored.

### Step 6: Confirm and commit

Show the user the proposed commit message and the list of files being committed. Use the interactive question tool to confirm:

- Option 1: Commit as-is
- Option 2: Edit the message (let the user provide a modified version)
- Option 3: Cancel

If confirmed, execute the commit using a HEREDOC for proper formatting:

```bash
git commit -m "$(cat <<'EOF'
feat(scope): description here
EOF
)"
```

### Step 7: Verify git account (multi-account)

Before pushing, verify the active GitHub account matches the repository directory:

```bash
gh auth status
```

Check the current directory against this mapping:

| Directory pattern | Expected user |
|---|---|
| `~/git/TomyQB/*` or `~/git/web3/*` | TomyQB |
| `~/git/MTDevops/*` | MTDevops |

If the active account doesn't match, run `gh auth switch` before pushing. If the directory doesn't match any known pattern, warn the user and ask which account to use.

### Step 8: Push with confirmation

Ask the user before pushing:

- Show the current branch
- Show the remote it will push to
- Ask for confirmation

If the branch has no upstream, use `git push -u origin <branch>`.

If the branch is `main` or `master`, **warn the user** that pushing directly to the main branch is risky and suggest creating a PR instead. Only push if they explicitly confirm.

After a successful push, show the result and the remote URL if available.

## Edge cases

- **Pre-commit hooks fail**: If the commit fails due to hooks, read the error output, fix the issue, re-stage, and create a **new** commit (never `--amend` after a hook failure, as the commit didn't happen).
- **Merge conflicts**: Don't commit. Tell the user to resolve conflicts first.
- **Empty diff but staged files**: Proceed normally with the staged files.
- **Detached HEAD**: Warn the user and suggest creating a branch first.
