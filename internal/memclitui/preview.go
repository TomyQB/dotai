package memclitui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// PreviewModel renders a file inside a scrollable viewport. Content is loaded
// eagerly at construction time so resize does not trigger a re-read.
type PreviewModel struct {
	vp      viewport.Model
	path    string
	body    string
	ready   bool
	width   int
	height  int
}

// NewPreview returns a PreviewModel showing the contents of path. Read errors
// are rendered inside the viewport rather than propagated — the caller
// stays responsible only for navigation.
func NewPreview(path string) PreviewModel {
	return PreviewModel{
		path: path,
		body: readBody(path),
	}
}

// Title satisfies Screen.
func (m PreviewModel) Title() string { return filepath.Base(m.path) }

// Init satisfies tea.Model.
func (m PreviewModel) Init() tea.Cmd { return nil }

// Update handles keyboard input and resizes. All navigation keys except
// scroll-up/down/page are treated as "back".
func (m PreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureViewport()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "left", "q":
			return m, popCmd()
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// ensureViewport creates or resizes the viewport on the first WindowSizeMsg.
func (m *PreviewModel) ensureViewport() {
	w := m.width - 6
	if w < 10 {
		w = 10
	}
	h := m.height - 8
	if h < 3 {
		h = 3
	}
	if !m.ready {
		m.vp = viewport.New(w, h)
		m.vp.SetContent(m.body)
		m.ready = true
		return
	}
	m.vp.Width = w
	m.vp.Height = h
}

// SetSize stores the terminal dimensions and reshapes the viewport.
func (m *PreviewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.ensureViewport()
}

// View renders the preview in a compact frame with scroll/back footer.
func (m PreviewModel) View() string {
	header := banner.RenderCompact(styles.CompactInnerWidth, filepath.Base(m.path))
	body := m.vp.View()
	footer := styles.FooterHints(
		"↑/↓", "scroll",
		"esc/←", "back",
	)
	return styles.FrameCompact(header, body, footer, styles.CompactWidth)
}

// readBody loads the file contents. On error it returns a human-readable
// message so the viewport never ends up blank.
func readBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading %s: %v", path, err)
	}
	return string(data)
}
