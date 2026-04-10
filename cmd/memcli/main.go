// Package main is the memcli entry point. All os.Exit calls live here; library
// packages never exit.
package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/TomyQB/mem-cli/internal/tui"
	"github.com/TomyQB/mem-cli/internal/version"
)

func usage() {
	fmt.Println(`mem-cli v` + version.Version + `

Usage:
  memcli      Launch interactive TUI (requires a terminal)

All actions (install, doctor, browse, prune) live inside the TUI.
Run memcli in a terminal to get started.`)
}

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Println(version.Version)
			return
		case "help", "-h", "--help":
			usage()
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q — run memcli to launch the TUI\n", os.Args[1])
			os.Exit(1)
		}
	}

	// No args: TUI if stdout is a TTY, otherwise print usage.
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println("memcli — run in a terminal to launch the TUI")
		return
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
