// Package memclitui implements the interactive TUI for the `memcli` binary:
// a read-only browser of registered projects and their .agent-memory/ trees.
//
// The screen hierarchy is a classic stack:
//
//	projects → browser(depth 0..N) → preview
//
// Navigation contract is the same one used throughout dotai:
//
//	↑/↓       move cursor
//	Enter     enter directory / open file
//	→         same as Enter
//	←         go back one level (up one dir, or out of preview)
//	Esc       pop the whole screen (back to the previous screen in the stack)
//	q / ^C    quit the program
package memclitui

import tea "github.com/charmbracelet/bubbletea"

// Screen is the contract every screen in the memcli TUI implements.
type Screen interface {
	tea.Model
	Title() string
}

// PopScreenMsg asks the root to pop the top screen from the stack.
type PopScreenMsg struct{}

// PushScreenMsg asks the root to push a new screen onto the stack.
type PushScreenMsg struct {
	Screen Screen
}
