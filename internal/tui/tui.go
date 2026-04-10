// Package tui is the Bubbletea interactive shell for memcli. Run() builds the
// root model (screen stack + optional modal overlay) and starts the event loop.
// Library code NEVER calls os.Exit; errors are returned to cmd/memcli/main.go.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/mem-cli/internal/install"
	"github.com/TomyQB/mem-cli/internal/registry"
	"github.com/TomyQB/mem-cli/internal/tui/home"
	"github.com/TomyQB/mem-cli/internal/tui/menu"
	"github.com/TomyQB/mem-cli/internal/tui/messages"
	"github.com/TomyQB/mem-cli/internal/tui/preview"
)

// Run starts the interactive TUI on the Menu screen. Returns any non-nil error
// from the Bubbletea program so the caller can decide how to surface it.
func Run() error {
	reg, err := registry.Load()
	if err != nil {
		// Non-fatal: start with an empty registry but surface the error
		// to the user via the TUI's error display.
		reg = &registry.Registry{Version: registry.SchemaVersion}
		fmt.Println("warning: failed to load registry:", err)
	}
	root := newRootModel(reg)
	p := tea.NewProgram(root)
	_, err = p.Run()
	return err
}

// rootModel holds the screen stack and optional modal overlay. All screen
// navigation is stack-based: enter pushes, esc pops, empty pop on root quits.
type rootModel struct {
	stack  []messages.Screen
	width  int
	height int
	reg    *registry.Registry
	modal  tea.Model
}

func newRootModel(reg *registry.Registry) rootModel {
	return rootModel{
		stack: []messages.Screen{menu.New()},
		reg:   reg,
	}
}

func (m rootModel) Init() tea.Cmd {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1].Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Broadcast to every screen on the stack so background screens
		// are sized correctly when the user returns.
		for i := range m.stack {
			updated, _ := m.stack[i].Update(msg)
			if s, ok := updated.(messages.Screen); ok {
				m.stack[i] = s
			}
		}
		if m.modal != nil {
			updated, _ := m.modal.Update(msg)
			m.modal = updated
		}
		return m, nil

	case menu.SelectedMsg:
		return m.handleMenuAction(msg.Action)

	case messages.ProjectSelectedMsg:
		return m.pushProject(msg), nil

	case messages.DocSelectedMsg:
		return m.pushPreview(msg)

	case messages.DoctorResultMsg:
		m.modal = m.openModal(newDoctorModal(msg.Report, nil))
		return m, nil

	case messages.DoctorErrMsg:
		m.modal = m.openModal(newDoctorModal(messages.EmptyReport(), msg.Err))
		return m, nil

	case messages.PopScreenMsg:
		m, cmd := m.pop()
		return m, cmd

	case messages.DismissModalMsg:
		m.modal = nil
		return m, nil

	case tea.KeyMsg:
		// Modal captures all keys when set.
		if m.modal != nil {
			switch msg.String() {
			case "esc", "q":
				m.modal = nil
				return m, nil
			}
			updated, cmd := m.modal.Update(msg)
			m.modal = updated
			return m, cmd
		}
		// Global quit.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Global esc: pop if not at root.
		if msg.String() == "esc" && len(m.stack) > 1 {
			m, cmd := m.pop()
			return m, cmd
		}
	}

	// Route everything else to top of stack.
	if len(m.stack) == 0 {
		return m, tea.Quit
	}
	top := m.stack[len(m.stack)-1]
	updated, cmd := top.Update(msg)
	if s, ok := updated.(messages.Screen); ok {
		m.stack[len(m.stack)-1] = s
	}
	return m, cmd
}

func (m rootModel) View() string {
	if len(m.stack) == 0 {
		return "goodbye\n"
	}
	if m.modal != nil {
		return m.modal.View()
	}
	return m.stack[len(m.stack)-1].View()
}

// openModal forwards the current window size to a newly opened modal so its
// viewport and placement are sized correctly on first render.
func (m rootModel) openModal(modal tea.Model) tea.Model {
	if m.width > 0 && m.height > 0 {
		updated, _ := modal.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return updated
	}
	return modal
}

// pop removes the top screen from the stack. If the popped screen is a Preview,
// it returns tea.ExitAltScreen to leave fullscreen mode.
func (m rootModel) pop() (rootModel, tea.Cmd) {
	if len(m.stack) <= 1 {
		return m, nil
	}
	popped := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	if isPreview(popped) {
		return m, tea.ExitAltScreen
	}
	return m, nil
}

// isPreview returns true if the screen is a preview.Model (value or pointer).
func isPreview(s messages.Screen) bool {
	switch s.(type) {
	case preview.Model, *preview.Model:
		return true
	}
	return false
}

// handleMenuAction wires Menu selections to their corresponding effects.
func (m rootModel) handleMenuAction(action menu.Action) (tea.Model, tea.Cmd) {
	switch action {
	case menu.ActionInstall:
		return m, runInstallCmd()
	case menu.ActionBrowseProjects:
		m = m.pushScreen(home.New(m.reg))
		return m, nil
	case menu.ActionDoctorCWD:
		cwd, err := os.Getwd()
		if err != nil {
			return m, func() tea.Msg { return messages.DoctorErrMsg{Err: err} }
		}
		return m, messages.RunDoctorCmd(cwd)
	case menu.ActionPruneRegistry:
		return m, runPruneCmd()
	case menu.ActionQuit:
		return m, tea.Quit
	}
	return m, nil
}

// runInstallCmd executes install.Run asynchronously and returns a toast.
func runInstallCmd() tea.Cmd {
	return func() tea.Msg {
		if err := install.Run(); err != nil {
			return menu.ToastMsg{Text: "Install failed: " + err.Error()}
		}
		return menu.ToastMsg{Text: "Installed successfully"}
	}
}

// runPruneCmd executes registry.Prune asynchronously and returns a toast with
// the number of removed entries. Errors are surfaced as a toast too.
func runPruneCmd() tea.Cmd {
	return func() tea.Msg {
		reg, err := registry.Load()
		if err != nil {
			return menu.ToastMsg{Text: "prune failed: " + err.Error()}
		}
		removed := reg.Prune()
		if removed == 0 {
			return menu.ToastMsg{Text: "Nothing to prune"}
		}
		if err := reg.Save(); err != nil {
			return menu.ToastMsg{Text: "prune save failed: " + err.Error()}
		}
		return menu.ToastMsg{Text: fmt.Sprintf("Pruned %d entries", removed)}
	}
}
