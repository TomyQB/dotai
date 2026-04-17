// Package app implements the root Bubble Tea model for dotai.
// It owns a screen stack and routes messages to the top-most screen,
// handling push/pop navigation messages and global key bindings.
package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/doctor"
	"github.com/TomyQB/dotai/internal/tui/memclipanel"
	"github.com/TomyQB/dotai/internal/tui/menu"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/status"
	"github.com/TomyQB/dotai/internal/tui/updatescreen"
	"github.com/TomyQB/dotai/internal/wizard"
)

// StepFactory is a function that creates a fresh wizard.Step.
// Factories are used instead of pre-built steps so each wizard invocation
// starts with clean state (no stale confirmed/cursor values from prior runs).
type StepFactory func() wizard.Step

// Model is the root Bubble Tea model. It maintains a screen stack and
// delegates rendering and input handling to the top-most screen.
type Model struct {
	stack     []messages.Screen
	prov      provider.Provider
	factories []StepFactory
	width     int
	height    int
}

// New constructs a Model with the main menu as the initial screen.
// factories are stored so a fresh wizard with clean steps is created on each
// ActionInstall dispatch.
func New(prov provider.Provider, factories []StepFactory) Model {
	m := menu.New()
	return Model{
		stack:     []messages.Screen{m},
		prov:      prov,
		factories: factories,
	}
}

// NewResumedUpdate returns a Model whose initial stack is [menu, update], with
// the update screen pre-entered in "resumed" mode. This is used after the
// binary relaunches itself post-`brew upgrade`, so the freshly-exec'd process
// continues with the config-file re-apply phase without making the user
// navigate there again.
func NewResumedUpdate(prov provider.Provider, factories []StepFactory) Model {
	m := Model{
		stack:     []messages.Screen{menu.New()},
		prov:      prov,
		factories: factories,
	}
	m.push(updatescreen.NewResumed(prov))
	return m
}

// Init satisfies tea.Model — delegates to the initial screen.
func (m Model) Init() tea.Cmd {
	return m.stack[0].Init()
}

// Update handles global messages first, then routes to the top screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global handlers — evaluated before any screen-specific logic.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to every screen via Update. broadcastSize cannot reach
		// pointer-receiver SetSize methods when the interface contains a
		// value (every New* returns by value), whereas Update is part of
		// tea.Model itself and resolves pointer receivers transparently.
		for i, s := range m.stack {
			updated, _ := s.Update(msg)
			if sc, ok := updated.(messages.Screen); ok {
				m.stack[i] = sc
			}
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case messages.PopScreenMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, m.top().Init()
		}
		return m, nil

	case messages.PushScreenMsg:
		m.stack = append(m.stack, msg.Screen)
		m.deliverSize()
		return m, m.top().Init()

	case menu.SelectedMsg:
		return m.handleMenuAction(msg.Action)
	}

	// Default: forward the message to the active screen.
	updated, cmd := m.top().Update(msg)
	if s, ok := updated.(messages.Screen); ok {
		m.stack[len(m.stack)-1] = s
	}
	return m, cmd
}

// View delegates rendering to the top-most screen.
func (m Model) View() string {
	return m.top().View()
}

// top returns the active (top-most) screen from the stack.
func (m Model) top() messages.Screen {
	return m.stack[len(m.stack)-1]
}

// push appends a screen to the stack and delivers the current window size.
func (m *Model) push(s messages.Screen) {
	m.stack = append(m.stack, s)
	m.deliverSize()
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
	if sc, ok := updated.(messages.Screen); ok {
		m.stack[len(m.stack)-1] = sc
	}
}

// handleMenuAction dispatches a menu selection to the appropriate screen.
func (m Model) handleMenuAction(action menu.Action) (tea.Model, tea.Cmd) {
	switch action {
	case menu.ActionInstall:
		freshSteps := make([]wizard.Step, len(m.factories))
		for i, f := range m.factories {
			freshSteps[i] = f()
		}
		ws := newWizardScreen(m.prov, freshSteps)
		m.push(ws)
		return m, ws.Init()

	case menu.ActionUpdate:
		s := updatescreen.New(m.prov)
		m.push(s)
		return m, s.Init()

	case menu.ActionStatus:
		s := status.New(m.prov)
		m.push(s)
		return m, s.Init()

	case menu.ActionMemcli:
		s := memclipanel.New(m.prov)
		m.push(s)
		return m, s.Init()

	case menu.ActionDoctor:
		s := doctor.New(m.prov)
		m.push(s)
		return m, s.Init()

	case menu.ActionQuit:
		return m, tea.Quit
	}

	return m, nil
}

