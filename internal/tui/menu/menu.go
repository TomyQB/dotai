// Package menu implements the main menu screen for dotai.
// It renders the dotai banner followed by a list of navigable actions.
package menu

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// Action identifies a menu item choice.
type Action int

const (
	ActionInstall Action = iota
	ActionUpdate
	ActionStatus
	ActionMemcli
	ActionDoctor
	ActionQuit
)

// SelectedMsg is emitted when the user confirms a menu item.
type SelectedMsg struct{ Action Action }

// menuItem groups the data for a single menu row.
type menuItem struct {
	action Action
	label  string
	desc   string
}

// Model is the Bubble Tea model for the main menu screen.
type Model struct {
	items         []menuItem
	cursor        int
	width, height int
}

// New returns an initialised menu Model.
func New() Model {
	return Model{
		items: []menuItem{
			{ActionInstall, "Install", "Setup wizard"},
			{ActionUpdate, "Update", "Re-apply to installed"},
			{ActionStatus, "Status", "What's installed"},
			{ActionMemcli, "Memcli", "Manage memcli"},
			{ActionDoctor, "Doctor", "Diagnose issues"},
			{ActionQuit, "Quit", ""},
		},
	}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "dotai" }

// Init satisfies tea.Model — no I/O needed on entry.
func (m Model) Init() tea.Cmd { return nil }

// Update handles key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
		case "down", "j":
			m.cursor = (m.cursor + 1) % len(m.items)
		case "enter", " ":
			selected := m.items[m.cursor]
			if selected.action == ActionQuit {
				return m, tea.Quit
			}
			return m, func() tea.Msg { return SelectedMsg{Action: selected.action} }
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// SetSize stores the terminal dimensions (satisfies messages.Screen).
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the menu inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth

	header := banner.Render(innerWidth) + "\n"

	var body string
	for i, item := range m.items {
		var row string
		if i == m.cursor {
			row = styles.Arrow.Render("▸ ") +
				styles.MenuItemSelected.Render(item.label)
			if item.desc != "" {
				row += "  " + styles.Footer.Render(item.desc)
			}
		} else {
			row = "  " + styles.MenuItem.Render(item.label)
			if item.desc != "" {
				row += "  " + styles.Footer.Render(item.desc)
			}
		}
		if i < len(m.items)-1 {
			body += row + "\n"
		} else {
			body += row
		}
	}

	footer := m.Footer()
	return styles.FrameCompact(header, body, footer, styles.CompactWidth)
}

// Footer returns the key-hint line for the menu.
func (m Model) Footer() string {
	return styles.FooterHints("↑/↓", "navigate", "enter", "select", "q", "quit")
}
