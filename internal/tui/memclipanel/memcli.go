// Package memclipanel implements the memcli management screen for dotai.
// It shows the current memcli install status and allows the user to
// install/reinstall or uninstall memcli components in one action.
package memclipanel

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/wizard"
)

// statusMsg carries the result of the memcli component check.
type statusMsg struct{ installed bool }

// actionDoneMsg signals that an install or uninstall operation has completed.
type actionDoneMsg struct {
	err    error
	action string
}

// Model is the Bubble Tea model for the memcli management screen.
type Model struct {
	prov          provider.Provider
	installed     bool
	loaded        bool
	busy          bool
	cursor        int
	toast         string
	toastErr      bool
	width, height int
}

// New returns an initialised memclipanel Model for the given provider.
func New(prov provider.Provider) Model {
	return Model{prov: prov}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "memcli" }

// Init triggers an async check of the memcli component status.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		result := detect.CheckComponent(m.prov, detect.Memcli)
		return statusMsg{installed: result.Installed}
	}
}

// Update handles incoming messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusMsg:
		m.installed = msg.installed
		m.loaded = true
		return m, nil

	case actionDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.toast = msg.action + " failed: " + msg.err.Error()
			m.toastErr = true
		} else {
			m.toast = msg.action + " completed successfully"
			m.toastErr = false
		}
		// Re-check status after the operation to keep the view accurate.
		return m, m.Init()

	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 2 {
				m.cursor++
			}
		case "enter":
			switch m.cursor {
			case 0: // Install / Reinstall
				m.busy = true
				m.toast = ""
				return m, installCmd(m.prov)
			case 1: // Uninstall (only when installed)
				if m.installed {
					m.busy = true
					m.toast = ""
					return m, uninstallCmd(m.prov)
				}
			case 2: // Back
				return m, func() tea.Msg { return messages.PopScreenMsg{} }
			}
		case "esc", "q":
			return m, func() tea.Msg { return messages.PopScreenMsg{} }
		}
	}
	return m, nil
}

// SetSize stores the terminal dimensions (satisfies messages.Screen).
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the memcli panel inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth

	header := banner.RenderCompact(innerWidth, "memcli")

	var body string

	// Status line.
	if !m.loaded {
		body += styles.Help.Render("Checking status...") + "\n\n"
	} else {
		statusIcon := styles.CheckOff.Render("✗")
		statusText := styles.Help.Render("Not installed")
		if m.installed {
			statusIcon = styles.CheckOn.Render("✓")
			statusText = styles.MenuItem.Render("Installed")
		}
		body += "Status: " + statusIcon + " " + statusText + "\n\n"
	}

	// Working indicator.
	if m.busy {
		body += styles.WarningText.Render("Working...") + "\n\n"
	}

	// Action list.
	labels := []string{"Install", "Uninstall", "Back"}
	if m.installed {
		labels[0] = "Reinstall"
	}

	for i, label := range labels {
		var row string
		disabled := i == 1 && !m.installed

		if i == m.cursor {
			row = styles.Arrow.Render("▸ ") + styles.MenuItemSelected.Render(label)
		} else if disabled {
			row = "  " + styles.CheckOff.Render(label)
		} else {
			row = "  " + styles.MenuItem.Render(label)
		}

		if i < len(labels)-1 {
			body += row + "\n"
		} else {
			body += row
		}
	}

	// Toast message.
	if m.toast != "" {
		body += "\n"
		if m.toastErr {
			body += styles.ProgressError.Render(m.toast)
		} else {
			body += styles.Toast.Render(m.toast)
		}
	}

	return styles.FrameCompact(header, body, m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for this screen.
func (m Model) Footer() string {
	return styles.FooterHints("↑/↓", "navigate", "enter", "select", "esc", "back")
}

// installCmd runs InstallMemcli asynchronously and returns an actionDoneMsg.
func installCmd(prov provider.Provider) tea.Cmd {
	return func() tea.Msg {
		err := wizard.InstallMemcli(prov)
		return actionDoneMsg{err: err, action: "install"}
	}
}

// uninstallCmd runs UninstallMemcli asynchronously and returns an actionDoneMsg.
func uninstallCmd(prov provider.Provider) tea.Cmd {
	return func() tea.Msg {
		err := wizard.UninstallMemcli(prov)
		return actionDoneMsg{err: err, action: "uninstall"}
	}
}
