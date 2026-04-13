---
name: readme-generator
description: Genera READMEs profesionales y visualmente atractivos con badges centrados, features, API/interface tables, project structure, usage y testing. Analiza automáticamente el código fuente, tests, configs y dependencias del proyecto para generar todo el contenido sin preguntar. Usa este skill siempre que el usuario pida crear un README, generar un README, actualizar el README, "hazme un readme bonito", "crea el readme", "genera documentación del proyecto", "write a README", "create a readme", "update the readme", "readme profesional", o cualquier variación. También se activa cuando el usuario dice cosas como "documenta el proyecto", "necesito un readme", "el readme está vacío", o "pon un readme como los otros proyectos".
---

# Professional README Generator

Generate polished, developer-focused READMEs by analyzing the project's actual code, tests, and configuration. The README should feel like it was written by a senior developer who knows the project inside out — concise, accurate, and visually clean.

## Why this style works

Developers scan READMEs, they don't read them. The centered header with badges gives instant context about the tech stack. The features section tells them if the project does what they need. The interface table is the quick reference they'll come back to. Everything else supports getting started fast.

## Step 1: Analyze the project

Before writing anything, understand what you're documenting. Read these in parallel:

1. **Config files** — detect the stack:
   - `foundry.toml` → Solidity + Foundry
   - `package.json` → Node.js (check for framework: Next.js, Express, React, etc.)
   - `Cargo.toml` → Rust
   - `pyproject.toml` / `requirements.txt` → Python
   - `go.mod` → Go
   - `pom.xml` / `build.gradle` → Java/Kotlin
   - Also check for: Docker, CI/CD configs, `.env.example`

2. **Source code** — read all files in `src/`, `app/`, `lib/`, or equivalent. Extract:
   - Public API: functions, endpoints, contract interfaces, CLI commands
   - Key features and patterns used
   - Access control or auth patterns

3. **Tests** — read test files to understand coverage scope. Count deterministic vs fuzz/property tests.

4. **Dependencies** — check for notable libraries (OpenZeppelin, Express, React, Django, etc.)

5. **Deploy/scripts** — check for deployment scripts, Makefiles, docker-compose

6. **Problem domain** — from the code and config, determine:
   - What problem the project solves (look at the main entry point, route handlers, core logic, or contract purpose)
   - How it solves it (the approach: what pattern, protocol, or strategy the code implements)
   - The key functional flows a user or consumer would care about (not implementation details, but outcomes)
   - If the project is a generic utility or boilerplate, describe the workflow it enables rather than fabricating a "problem"

## Step 2: Generate the README

Use this exact structure. Every section is required unless marked optional.

### Header (centered)

```markdown
<div align="center">

![Badge1](url) ![Badge2](url) ...

# Project Title

One-sentence description of what the project does, what it's built with, and its key capability.

</div>
```

**Badge rules:**
- Use `shields.io` with `style=for-the-badge`
- Format: `https://img.shields.io/badge/<LABEL>-<VALUE>-<COLOR>?style=for-the-badge&logo=<LOGO>&logoColor=white`
- Include badges for: primary language + version, framework/toolchain, major dependencies, license
- Keep to 3-5 badges max — only the most important

**Common badge templates:**

| Stack | Badge |
|-------|-------|
| Solidity | `![Solidity](https://img.shields.io/badge/Solidity-^0.8.28-363636?style=for-the-badge&logo=solidity&logoColor=white)` |
| Foundry | `![Foundry](https://img.shields.io/badge/Foundry-Forge-DEA584?style=for-the-badge&logo=ethereum&logoColor=white)` |
| OpenZeppelin | `![OpenZeppelin](https://img.shields.io/badge/OpenZeppelin-v5.x-4E5EE4?style=for-the-badge&logo=openzeppelin&logoColor=white)` |
| TypeScript | `![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6?style=for-the-badge&logo=typescript&logoColor=white)` |
| React | `![React](https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black)` |
| Next.js | `![Next.js](https://img.shields.io/badge/Next.js-14-000000?style=for-the-badge&logo=next.js&logoColor=white)` |
| Node.js | `![Node.js](https://img.shields.io/badge/Node.js-20-339933?style=for-the-badge&logo=node.js&logoColor=white)` |
| Python | `![Python](https://img.shields.io/badge/Python-3.12-3776AB?style=for-the-badge&logo=python&logoColor=white)` |
| Rust | `![Rust](https://img.shields.io/badge/Rust-1.75-DEA584?style=for-the-badge&logo=rust&logoColor=white)` |
| Go | `![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go&logoColor=white)` |
| Docker | `![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white)` |
| License MIT | `![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)` |

For badges not in this table, follow the same pattern. Use the actual version found in the project config, not a placeholder.

### Features

```markdown
## Features

- **Bold keyword**: Brief explanation of the feature
- **Another feature**: What it does and why it matters
```

- 4-7 bullet points
- Each starts with a **bold keyword** that can be scanned
- Focus on what makes this project interesting, not obvious boilerplate
- Use inline code for function/class names

### Overview

```markdown
## Overview

One paragraph (2-3 sentences) stating what problem the project addresses and the context in which it operates.

One paragraph (2-3 sentences) explaining how the project solves that problem. Focus on the approach and key design decisions — not implementation details, but the strategy.

One paragraph (1-2 sentences, optional) highlighting the key functional flows or primary use cases the end user experiences.
```

**Rules:**
- 2-3 short paragraphs, never more than 4. Total under ~120 words
- Prose, not bullets — this section complements the scannable Features list, it doesn't duplicate it
- Every claim must be derived from actual code analyzed in Step 1. Do not speculate
- Do not repeat the Features list in prose form. Overview = "why" and "how"; Features = "what"
- For small utilities or single-purpose tools, two paragraphs are enough — skip the third
- No sub-headers within Overview

### API / Interface (adapt to project type)

For smart contracts:
```markdown
## Contract Interface

| Function | Access | Description |
|---|---|---|
| `functionName(params)` | Public/Owner/Admin | What it does |
```

For REST APIs:
```markdown
## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/users` | List all users |
```

For CLI tools:
```markdown
## Commands

| Command | Description |
|---|---|
| `tool init` | Initialize a new project |
```

For libraries:
```markdown
## API

| Function | Description |
|---|---|
| `parse(input)` | Parses the input string |
```

If the project has multiple contracts/modules, use sub-headers for each.

### Project Structure

```markdown
## Project Structure

\```
src/
  MainFile.ext         # Brief description
test/
  MainFile.test.ext    # What's tested
lib/
  dependency/          # Version info
\```
```

- Only include top-level directories and key files
- Add `# comments` for context
- Don't list every file — focus on what matters

### Requirements

```markdown
## Requirements

- [Tool Name](install-url)
```

Only the essential tools needed to build and run.

### Usage

```markdown
## Usage

\```shell
# Build
build-command

# Run tests
test-command

# Other relevant commands
\```
```

- One code block with the most common commands
- Add comments for each command
- Include deploy command if a deploy script exists
- Don't use `$` prefix on commands

### Testing (optional — include if tests exist)

```markdown
## Testing

The test suite includes **N deterministic tests** and **M fuzz/property tests**:

- Brief description of test categories
- What's covered

\```shell
# Run with coverage/gas report
coverage-command
\```
```

Count the actual tests by reading the test files. Be specific about numbers.

## What NOT to include

- Table of contents (the README is short enough to scan)
- Contributing section (unless the project is open source and expects contributors)
- Changelog (that's what git log is for)
- Badges for things that aren't core to the stack (no "PRs welcome", no "code style", no build status unless CI exists)
- Screenshots (unless the project has a UI)
- Verbose explanations of obvious things

## Tone

- Technical but approachable
- Concise — if a sentence doesn't add value, cut it
- Accurate — every claim must match the actual code
- Professional — no emojis in the body, no exclamation marks, no hype
