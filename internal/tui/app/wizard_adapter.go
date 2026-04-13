package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/wizard"
)

// wizardScreen adapts wizard.WizardModel to the messages.Screen interface.
// Its primary responsibility is intercepting terminal signals from the wizard
// (tea.Quit on cancel, "any key" in PhaseDone/PhaseError) and converting them
// into a PopScreenMsg so that the user returns to the menu instead of quitting
// the entire program.
type wizardScreen struct {
	wiz wizard.WizardModel
}

// newWizardScreen creates a fresh wizardScreen for each ActionInstall dispatch.
// A fresh instance is always created — never reused — so the wizard state is
// clean on every visit.
func newWizardScreen(prov provider.Provider, steps []wizard.Step) wizardScreen {
	return wizardScreen{
		wiz: wizard.New(prov, steps),
	}
}

// Title satisfies messages.Screen.
func (w wizardScreen) Title() string { return "install" }

// Init satisfies tea.Model — delegates to the inner wizard.
func (w wizardScreen) Init() tea.Cmd {
	return w.wiz.Init()
}

// Update satisfies tea.Model.
//
// Key interceptions (in order):
//  1. Any key in PhaseDone or PhaseError: the wizard would normally call
//     tea.Quit here; we intercept BEFORE forwarding so the app stays alive.
//  2. After forwarding: tea.Quit emitted by the SummaryCancelMsg path is
//     wrapped by interceptQuit and converted to a PopScreenMsg.
func (w wizardScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Intercept key presses in terminal phases before the wizard handles them.
	// The wizard's own handler would return tea.Quit; we want PopScreenMsg.
	if _, ok := msg.(tea.KeyMsg); ok {
		phase := w.wiz.Phase()
		if phase == wizard.PhaseDone || phase == wizard.PhaseError {
			return w, func() tea.Msg { return messages.PopScreenMsg{} }
		}
	}

	// Forward to the inner wizard.
	updated, cmd := w.wiz.Update(msg)
	if wm, ok := updated.(wizard.WizardModel); ok {
		w.wiz = wm
	}

	// If the user cancelled at the summary screen, let tea.Quit propagate
	// so the program exits entirely (REQ-4.2). Only convert tea.Quit to
	// PopScreenMsg for the done/error completion paths.
	if !w.wiz.WasCancelled() {
		cmd = interceptQuit(cmd)
	}

	return w, cmd
}

// View satisfies tea.Model — delegates to the inner wizard.
func (w wizardScreen) View() string {
	return w.wiz.View()
}

// SetSize is a no-op: the wizard gets resized via the tea.WindowSizeMsg
// broadcast performed by app.Model.Update through the normal Update path.
func (w *wizardScreen) SetSize(_, _ int) {}

// Footer satisfies an optional convention used by some screens.
// The wizard manages its own footer inside its Frame rendering.
func (w wizardScreen) Footer() string { return "" }

// interceptQuit wraps a Cmd so that any tea.QuitMsg it produces is converted
// into a PopScreenMsg. This prevents the wizard's cancel action from exiting
// the entire program when it is embedded inside the app stack.
func interceptQuit(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			return messages.PopScreenMsg{}
		}
		return msg
	}
}
