// Package doctor implements the doctor diagnostic screen for dotai.
// It runs detect.CheckAll and also probes config-directory writability,
// rendering results as a badge-based table.
package doctor

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// diagMsg carries the detection results and config-dir writability back to the model.
type diagMsg struct {
	results  []detect.CheckResult
	writable bool
}

// Model is the Bubble Tea model for the doctor screen.
type Model struct {
	prov           provider.Provider
	results        []detect.CheckResult
	configWritable bool
	loaded         bool
	width, height  int
}

// New returns an initialised doctor Model for the given provider.
func New(prov provider.Provider) Model {
	return Model{prov: prov}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "doctor" }

// Init triggers an async full diagnostic run.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		results := detect.CheckAll(m.prov)

		writable := false
		configDir, err := m.prov.ConfigDir()
		if err == nil {
			tmp, tmpErr := os.CreateTemp(configDir, ".dotai-probe-*")
			if tmpErr == nil {
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
				writable = true
			}
		}

		return diagMsg{results: results, writable: writable}
	}
}

// Update handles incoming messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diagMsg:
		m.results = msg.results
		m.configWritable = msg.writable
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

// View renders the doctor screen inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth

	header := banner.RenderCompact(innerWidth, "doctor")

	if !m.loaded {
		return styles.FrameCompact(header, "Running diagnostics...", m.Footer(), styles.CompactWidth)
	}

	var body string

	for _, r := range m.results {
		body += renderRow(r.Component.String(), r.Installed, r.Detail) + "\n"
	}

	// 7th row: config directory writability (computed locally).
	configWritableDetail := "~/.claude/ is writable"
	if !m.configWritable {
		configWritableDetail = "~/.claude/ is not writable"
	}
	body += renderRow("Config writable", m.configWritable, configWritableDetail)

	return styles.FrameCompact(header, body, m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for this screen.
func (m Model) Footer() string {
	return styles.FooterHints("esc", "back")
}

// renderRow formats a single diagnostic row with badge, label, and detail.
func renderRow(label string, installed bool, detail string) string {
	var badge string
	if installed {
		badge = styles.BadgeOK()
	} else {
		badge = styles.BadgeMissing()
	}
	return styles.MenuItem.Render(label) +
		"  " + badge +
		"  " + styles.Dim(detail)
}
