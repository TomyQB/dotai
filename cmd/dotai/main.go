// Package main is the dotai entry point.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	xterm "github.com/charmbracelet/x/term"

	"github.com/TomyQB/dotai/internal/provider/claude"
	"github.com/TomyQB/dotai/internal/tui/app"
	"github.com/TomyQB/dotai/internal/version"
	"github.com/TomyQB/dotai/internal/wizard"
	"github.com/TomyQB/dotai/internal/wizard/steps"
)

func main() {
	// Handle --version/-v and --help/-h flags before touching the terminal.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Println("dotai v" + version.Version)
			return
		case "help", "-h", "--help":
			fmt.Println("dotai — AI coding environment setup wizard")
			fmt.Println("Usage: dotai [command]")
			fmt.Println("\nCommands:")
			fmt.Println("  version    Print version")
			fmt.Println("  help       Print this help")
			fmt.Println("\nRun with no arguments to launch the interactive menu.")
			return
		}
	}

	// Guard: refuse to run when stdout is not a real terminal (e.g. piped).
	if !xterm.IsTerminal(os.Stdout.Fd()) {
		fmt.Println("dotai — run in a terminal to launch the setup wizard")
		return
	}

	prov := claude.New()
	if _, err := prov.ConfigDir(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	// Build step factories so each wizard invocation gets fresh steps with clean
	// state. Factories are created here to avoid an import cycle
	// (wizard → wizard/steps → wizard is forbidden).
	factories := []app.StepFactory{
		func() wizard.Step { return steps.NewProfile(prov) },
		func() wizard.Step { return steps.NewSkills(prov) },
		func() wizard.Step { return steps.NewMemcli(prov) },
	}

	root := app.New(prov, factories)
	p := tea.NewProgram(root, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
