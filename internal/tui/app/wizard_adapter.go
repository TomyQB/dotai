package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/wizard"
)

// wizardScreen adapts wizard.WizardModel to the messages.Screen interface.
// Its single responsibility is converting wizard.WizardExitMsg into a
// messages.PopScreenMsg so the user always returns to the landing menu
// instead of the whole program exiting.
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

// Update satisfies tea.Model. Any WizardExitMsg bubbling up from the wizard
// (triggered by esc, left-at-first-step, summary cancel, or any-key after
// done/error) is translated into a PopScreenMsg so the app stays alive.
func (w wizardScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(wizard.WizardExitMsg); ok {
		return w, func() tea.Msg { return messages.PopScreenMsg{} }
	}

	updated, cmd := w.wiz.Update(msg)
	if wm, ok := updated.(wizard.WizardModel); ok {
		w.wiz = wm
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
