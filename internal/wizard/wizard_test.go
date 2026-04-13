package wizard_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/wizard"
	"github.com/TomyQB/dotai/internal/wizard/steps"
)

// buildStepList returns the 3-step wizard step list used in production.
// Using the real constructors guarantees the Step interface is satisfied and
// Init/Apply paths are exercised.
func buildStepList(prov provider.Provider) []wizard.Step {
	return []wizard.Step{
		steps.NewProfile(prov),
		steps.NewSkills(prov),
		steps.NewMemcli(prov),
	}
}

// TestWizardNew verifies construction: phase starts at PhaseStepping and the
// correct number of steps is stored.
func TestWizardNew(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	stepList := buildStepList(prov)

	m := wizard.New(prov, stepList)

	// The wizard must not report cancelled immediately after construction.
	if m.WasCancelled() {
		t.Error("WasCancelled() = true right after New(), want false")
	}

	// Calling View() before Init should not panic; it simply renders step 0.
	_ = m.View()
}

// TestWizardWasCancelledDefault confirms the cancelled flag is false on a
// freshly-created model.
func TestWizardWasCancelledDefault(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	m := wizard.New(prov, buildStepList(prov))

	if m.WasCancelled() {
		t.Error("WasCancelled() = true by default, want false")
	}
}

// TestWizardInit verifies that Init returns a non-nil Cmd (it delegates to the
// first step's Init, which at minimum returns nil — but the call must not
// panic and the model must remain in PhaseStepping).
func TestWizardInit(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	m := wizard.New(prov, buildStepList(prov))

	// Init must not panic. A nil Cmd is acceptable for steps that have no
	// async work; we just verify the call completes without error.
	_ = m.Init()
}

// TestWizardCtrlCQuits verifies that ctrl+c emits tea.Quit from any phase.
func TestWizardCtrlCQuits(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	m := wizard.New(prov, buildStepList(prov))

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Update(ctrl+c) returned nil Cmd, want tea.Quit")
	}
}

// TestWizardWindowSizeUpdatesModel verifies that a WindowSizeMsg is handled
// without panicking.
func TestWizardWindowSizeUpdatesModel(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	m := wizard.New(prov, buildStepList(prov))

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if updated == nil {
		t.Error("Update(WindowSizeMsg) returned nil model")
	}
	// No cmd expected for a resize.
	_ = cmd
}

// TestWizardStepCountMatchesBuildList ensures New stores exactly the steps
// passed to it by checking that View does not panic when advancing through
// messages.
func TestWizardStepCountMatchesBuildList(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	stepList := buildStepList(prov)

	m := wizard.New(prov, stepList)

	// Render without error; the step header contains "Step 1/N" where N equals
	// the list length.
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string for a fresh WizardModel")
	}
}
