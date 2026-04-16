// Package main is the memcli binary: a read-only browser for the .agent-memory/
// tree of every project registered with memcli.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	xterm "github.com/charmbracelet/x/term"

	"github.com/TomyQB/dotai/internal/memclitui"
	"github.com/TomyQB/dotai/internal/registry"
	"github.com/TomyQB/dotai/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Println("memcli v" + version.Version)
			return
		case "help", "-h", "--help":
			fmt.Println("memcli — browse the .agent-memory/ of your registered projects")
			fmt.Println("Usage: memcli [command]")
			fmt.Println("\nCommands:")
			fmt.Println("  version    Print version")
			fmt.Println("  help       Print this help")
			fmt.Println("\nRun with no arguments to launch the interactive TUI.")
			return
		}
	}

	if !xterm.IsTerminal(os.Stdout.Fd()) {
		fmt.Println("memcli — run in a terminal to launch the browser")
		return
	}

	reg, err := registry.Load()
	if err != nil {
		// Non-fatal: start with an empty registry and surface the warning.
		fmt.Fprintln(os.Stderr, "warning: failed to load registry:", err)
		reg = &registry.Registry{Version: registry.SchemaVersion}
	}

	root := memclitui.New(reg)
	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
