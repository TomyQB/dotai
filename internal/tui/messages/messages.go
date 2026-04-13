// Package messages defines the shared message types used across TUI screens.
// Extracting these into a leaf package breaks import cycles between the root
// model and individual screen packages.
package messages

import tea "github.com/charmbracelet/bubbletea"

// Screen is the contract every TUI screen implements.
type Screen interface {
	tea.Model
	Title() string
}

// PopScreenMsg asks the root to pop the top screen from the stack.
type PopScreenMsg struct{}

// PushScreenMsg requests the root model to push a new screen onto the stack.
type PushScreenMsg struct {
	Screen Screen
}
