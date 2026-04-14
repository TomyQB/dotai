<div align="center">

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Bubbletea](https://img.shields.io/badge/Bubbletea-TUI-FF75B7?style=for-the-badge&logo=charmbracelet&logoColor=white)
![Claude Code](https://img.shields.io/badge/Claude_Code-Provider-D97757?style=for-the-badge&logo=anthropic&logoColor=white)
![GoReleaser](https://img.shields.io/badge/GoReleaser-Homebrew-1E1E1E?style=for-the-badge&logo=homebrew&logoColor=white)

# dotai

Interactive terminal wizard that configures your AI coding environment — profiles, skills, status line, git hooks, and living agent docs — in one guided flow.

</div>

## Features

- **Landing menu**: single TUI entry point with Install, Status, Memcli, Doctor and Quit screens built on Bubbletea with stack-based navigation.
- **Profile-driven install**: pick `Personal` or `Work` and dotai lays down `settings.json`, the status line, the git branch hook and the matching `CLAUDE.md` in one step.
- **Global skills catalog**: multi-select checklist to install curated skills (`commit-and-push`, `owasp-audit`, `readme-generator`, `skill-creator`, `web3-audit`, `web3-review`, `java-review`, `java-spring-boot`, `junit-mockito`).
- **Memcli integration**: optional living `.agent-memory/` documentation — installs skills, the `doc-keeper` agent and Stop/pre-commit hooks that keep docs in sync with the code.
- **Provider-agnostic core**: `internal/provider` abstracts Claude Code today; the wizard, detect and assets layers are ready for other providers without structural changes.
- **Component detection**: `internal/detect` inspects the filesystem so the Status and Doctor screens show exactly what is installed, stale or missing.
- **Safe CLI**: refuses to launch the TUI when stdout is not a terminal (piped, redirected, CI), so it never emits ANSI escapes into logs.

## Overview

Setting up an AI coding workspace means editing JSON settings, dropping shell hooks, copying skill folders and keeping a `CLAUDE.md` in sync — tasks that are fiddly, easy to get wrong and hard to reproduce across machines or between personal and work contexts. dotai centralises that setup behind one binary.

The tool ships every asset embedded inside the Go binary (`internal/assets/files`) and renders a Bubbletea TUI that walks you through profile, skills and memcli decisions. Each step runs an idempotent executor against the active provider (`internal/provider/claude`), so re-running dotai converges the environment instead of duplicating state. The same provider abstraction powers Status and Doctor, which read back the installed components instead of trusting the wizard.

Typical flow: launch `dotai` → pick `Install` → choose Personal or Work → tick the skills you want → opt in or out of memcli → return to the menu to verify everything with `Status` or `Doctor`.

## Commands

| Command | Description |
|---|---|
| `dotai` | Launch the interactive landing menu (requires a TTY). |
| `dotai version` | Print the dotai version. |
| `dotai help` | Show usage and available commands. |

## Menu Screens

| Screen | Description |
|---|---|
| `Install` | Runs the setup wizard: Profile → Skills → Memcli. |
| `Status` | Lists every dotai component and reports `OK` / `MISSING`. |
| `Memcli` | Dedicated panel to install, uninstall or inspect the memcli integration. |
| `Doctor` | Diagnoses global and per-project state, flags drift or broken hooks. |
| `Quit` | Exit the TUI. |

## Wizard Steps

| Step | What it installs |
|---|---|
| `Profile` | `settings.json`, status-line script, git branch-check hook, `CLAUDE.md` (Personal or Work variant). |
| `Skills` | Selected entries from the curated catalog into `~/.claude/skills/`. |
| `Memcli` | `memcli-{init,update,scan,doctor}` skills, the `doc-keeper` agent and the Stop + pre-commit hooks that keep `.agent-memory/` fresh. |

## Memcli Workflow

When memcli is enabled, the environment enforces living documentation for each registered project:

1. You work normally with Claude Code on your repo.
2. The Stop hook inspects the working tree, filters noise (tests, styles, lockfiles…) and marks any doc whose `watches` frontmatter matches a changed path as stale in `.agent-memory/.stale`.
3. The pre-commit hook blocks the commit while `.stale` is non-empty.
4. Running `/memcli-update` dispatches the `doc-keeper` agent in parallel, one per stale doc, to regenerate each from the current source.
5. Once every doc is rebuilt, `.stale` is cleared and the commit proceeds.

Each `.md` under `.agent-memory/` must carry frontmatter with a `watches` list plus both functional (Mermaid `flowchart`/`stateDiagram`) and technical (`sequenceDiagram`) context.

## Project Structure

```
cmd/
  dotai/              # Binary entry point — flag parsing + Bubbletea bootstrap
internal/
  assets/             # Embedded profiles, skills, hooks, settings, memcli assets
  detect/             # Component detection for Status/Doctor screens
  provider/           # Provider abstraction (Claude Code implementation)
    claude/
  tui/
    app/              # Stack-based screen navigation shell
    menu/             # Landing menu screen
    status/           # Status screen
    memclipanel/      # Memcli management panel
    doctor/           # Doctor diagnostics screen
    banner/, styles/  # Shared rendering
  wizard/             # Step contract, executor, summary
    steps/            # Profile, Skills, Memcli step implementations
  version/            # Single source of truth for the version string
.goreleaser.yaml      # Cross-platform release pipeline (linux/darwin/windows, amd64/arm64)
```

## Requirements

- [Go 1.25+](https://go.dev/dl/) to build from source.
- A POSIX terminal emulator capable of rendering ANSI/true-color for the Bubbletea TUI.
- [Claude Code](https://docs.claude.com/en/docs/claude-code) installed locally — dotai writes into its `~/.claude/` configuration tree.

## Usage

```shell
# Build from source
go build -o ~/.local/bin/dotai ./cmd/dotai

# Launch the interactive menu
dotai

# Print version / help
dotai version
dotai help
```

## Testing

The project ships unit tests across the wizard engine, steps, detect layer and TUI models.

```shell
# Run the full test suite
go test ./...

# Vet + tests with race detector
go vet ./... && go test -race ./...
```

## Status

v0.2 — landing menu with Install, Status, Memcli and Doctor screens. Profile-driven installer, curated skills catalog and memcli living-docs integration. Provider layer abstracted around Claude Code with room for additional providers.
