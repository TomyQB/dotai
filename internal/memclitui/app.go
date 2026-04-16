package memclitui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/registry"
)

// Model is the root Bubble Tea model for the memcli TUI. It owns a screen
// stack and routes messages/keys to the top-most screen. Global handling is
// limited to window sizing, Ctrl+C, and Push/Pop navigation messages.
type Model struct {
	stack  []Screen
	width  int
	height int
}

// New returns a Model initialised with the projects list as the root screen.
func New(reg *registry.Registry) Model {
	return Model{
		stack: []Screen{NewProjects(reg)},
	}
}

// Init delegates to the root screen.
func (m Model) Init() tea.Cmd {
	return m.top().Init()
}

// Update handles global messages first, then forwards to the top screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		for _, s := range m.stack {
			broadcastSize(s, m.width, m.height)
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case PopScreenMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, m.top().Init()
		}
		return m, tea.Quit

	case PushScreenMsg:
		m.push(msg.Screen)
		return m, msg.Screen.Init()
	}

	updated, cmd := m.top().Update(msg)
	if s, ok := updated.(Screen); ok {
		m.stack[len(m.stack)-1] = s
	}
	return m, cmd
}

// View delegates rendering to the top-most screen.
func (m Model) View() string {
	return m.top().View()
}

// top returns the current top-of-stack screen.
func (m Model) top() Screen {
	return m.stack[len(m.stack)-1]
}

// push appends a screen and broadcasts the current window size.
func (m *Model) push(s Screen) {
	m.stack = append(m.stack, s)
	broadcastSize(s, m.width, m.height)
}

// popCmd is the canonical emitter for "exit current screen". Shared by every
// screen in this package so nobody reinvents the closure.
func popCmd() tea.Cmd {
	return func() tea.Msg { return PopScreenMsg{} }
}

// pushCmd is the canonical emitter for "enter this screen".
func pushCmd(s Screen) tea.Cmd {
	return func() tea.Msg { return PushScreenMsg{Screen: s} }
}

// broadcastSize invokes SetSize on a screen that exposes it — local interface
// assertion so Screen does not need the extra method.
func broadcastSize(s Screen, w, h int) {
	type sizer interface {
		SetSize(w, h int)
	}
	if sz, ok := s.(sizer); ok {
		sz.SetSize(w, h)
	}
}
