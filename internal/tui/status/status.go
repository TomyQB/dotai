// Package status implements the status screen for dotai.
// It runs detect.CheckAll and renders one row per component.
package status

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// resultsMsg carries the detection results back to the model.
type resultsMsg []detect.CheckResult

// Model is the Bubble Tea model for the status screen.
type Model struct {
	prov          provider.Provider
	results       []detect.CheckResult
	loaded        bool
	width, height int
}

// New returns an initialised status Model for the given provider.
func New(prov provider.Provider) Model {
	return Model{prov: prov}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "status" }

// Init triggers an async check of all components.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		return resultsMsg(detect.CheckAll(m.prov))
	}
}

// Update handles incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultsMsg:
		m.results = []detect.CheckResult(msg)
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
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

// View renders the status screen inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth

	header := banner.RenderCompact(innerWidth, "status")

	if !m.loaded {
		return styles.FrameCompact(header, "Loading...", m.Footer(), styles.CompactWidth)
	}

	var body string
	for i, r := range m.results {
		name := r.Component.String()
		var row string
		if r.Installed {
			row = styles.CheckOn.Render("✓") + " " +
				styles.MenuItem.Render(name) +
				"  " + styles.Dim(r.Detail)
		} else {
			row = styles.CheckOff.Render("✗") + " " +
				styles.MenuItem.Render(name) +
				"  " + styles.Dim(r.Detail)
		}
		if i < len(m.results)-1 {
			body += row + "\n"
		} else {
			body += row
		}
	}

	return styles.FrameCompact(header, body, m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for the status screen.
func (m Model) Footer() string {
	return styles.FooterHints("esc", "back")
}
