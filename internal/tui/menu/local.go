// Package menu — local.go adds the menu variant used by `dotai local`.
// The local menu operates on the current working directory's .claude/ folder
// and therefore only exposes per-project actions (install/uninstall/status/
// doctor), hiding global-only items such as Memcli and Update.
package menu

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// LocalAction identifies a local-menu item choice.
type LocalAction int

const (
	LocalActionInstall LocalAction = iota
	LocalActionUninstall
	LocalActionStatus
	LocalActionDoctor
	LocalActionQuit
)

// LocalSelectedMsg is emitted when the user confirms a local-menu item.
type LocalSelectedMsg struct{ Action LocalAction }

// localMenuItem groups the data for a single local-menu row.
type localMenuItem struct {
	action LocalAction
	label  string
	desc   string
}

// LocalModel is the Bubble Tea model for the `dotai local` landing menu.
type LocalModel struct {
	items         []localMenuItem
	cursor        int
	targetDir     string
	width, height int
}

// NewLocal returns an initialised LocalModel for the given target directory.
// targetDir is shown beneath the menu so the user sees which project will be
// affected by Install / Uninstall.
func NewLocal(targetDir string) LocalModel {
	return LocalModel{
		targetDir: targetDir,
		items: []localMenuItem{
			{LocalActionInstall, "Install", "Add a profile to this repo"},
			{LocalActionUninstall, "Uninstall", "Remove a profile from this repo"},
			{LocalActionStatus, "Status", "What's installed here"},
			{LocalActionDoctor, "Doctor", "Diagnose issues"},
			{LocalActionQuit, "Quit", ""},
		},
	}
}

// Title satisfies messages.Screen.
func (m LocalModel) Title() string { return "dotai local" }

// Init satisfies tea.Model — no I/O needed on entry.
func (m LocalModel) Init() tea.Cmd { return nil }

// Update handles key events.
func (m LocalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
		case "down", "j":
			m.cursor = (m.cursor + 1) % len(m.items)
		case "enter", " ":
			selected := m.items[m.cursor]
			if selected.action == LocalActionQuit {
				return m, tea.Quit
			}
			return m, func() tea.Msg { return LocalSelectedMsg{Action: selected.action} }
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// SetSize stores the terminal dimensions (satisfies messages.Screen).
func (m *LocalModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the menu inside a FrameCompact border.
func (m LocalModel) View() string {
	innerWidth := styles.CompactInnerWidth

	header := banner.RenderCompact(innerWidth, "local") + "\n"

	var body string
	if m.targetDir != "" {
		body += styles.Help.Render("target: "+m.targetDir) + "\n\n"
	}
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

	return styles.FrameCompact(header, body, m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for the menu.
func (m LocalModel) Footer() string {
	return styles.FooterHints("↑/↓", "navigate", "enter", "select", "q", "quit")
}
