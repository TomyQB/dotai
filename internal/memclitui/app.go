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
		// Forward the size through every screen's Update. We cannot rely on
		// a side-channel SetSize interface here: New* helpers return models
		// by value, so the interface we hold contains a value — pointer-
		// receiver SetSize methods are invisible to a type assertion in that
		// case. Update IS on tea.Model itself, so pointer receivers resolve
		// transparently through the existing value→pointer copy-and-return
		// dance each screen already uses.
		for i, s := range m.stack {
			updated, _ := s.Update(msg)
			if sc, ok := updated.(Screen); ok {
				m.stack[i] = sc
			}
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
		m.stack = append(m.stack, msg.Screen)
		m.deliverSize()
		return m, m.top().Init()
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

// deliverSize hands the current window dimensions to the top-of-stack screen
// via a tea.WindowSizeMsg. Used right after every push so freshly-appended
// screens receive sizing immediately (Bubbletea does not re-emit
// WindowSizeMsg after the initial one). Update is the only channel that
// reaches pointer-receiver handlers when Screen holds a value.
func (m *Model) deliverSize() {
	if m.width <= 0 {
		return
	}
	sz := tea.WindowSizeMsg{Width: m.width, Height: m.height}
	updated, _ := m.top().Update(sz)
	if sc, ok := updated.(Screen); ok {
		m.stack[len(m.stack)-1] = sc
	}
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
