// Package preview implements the Preview screen: a scrollable viewport that
// renders a markdown file through Glamour, falling back to raw contents on
// render errors. File read errors are rendered in the viewport — never panic.
package preview

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/TomyQB/mem-cli/internal/tui/banner"
	"github.com/TomyQB/mem-cli/internal/tui/styles"
)

// Model is the Preview screen state.
type Model struct {
	vp       viewport.Model
	path     string
	rendered string // cached at construction — no re-render on resize
	width    int
	height   int
}

// New creates a Preview screen for the file at path.
func New(path string) Model {
	rendered := renderBody(path)
	vp := viewport.New(0, 0)
	vp.SetContent(rendered)
	return Model{vp: vp, path: path, rendered: rendered}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return m.path }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vp.Width = max(10, msg.Width-10)
		m.vp.Height = max(3, msg.Height-10)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// View renders the Preview screen inside a fullscreen Frame.
func (m Model) View() string {
	header := banner.RenderCompact(max(20, m.width-6), filepath.Base(m.path))
	footer := styles.FooterHints(
		"↑/↓", "scroll",
		"esc", "back",
		"q", "quit",
	)
	return styles.Frame(header, m.vp.View(), footer, m.width, m.height)
}

// renderBody reads the file and returns body text to place in the viewport.
// Errors are rendered as error text rather than propagated.
func renderBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading %s: %v", path, err)
	}
	rendered, err := glamour.Render(string(data), "auto")
	if err != nil {
		return string(data)
	}
	return rendered
}
