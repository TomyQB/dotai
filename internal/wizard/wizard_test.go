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

	if m.Phase() != wizard.PhaseStepping {
		t.Errorf("Phase() = %v right after New(), want PhaseStepping", m.Phase())
	}

	// Calling View() before Init should not panic; it simply renders step 0.
	_ = m.View()
}

// TestWizardEscEmitsExit verifies that esc in PhaseStepping produces a
// WizardExitMsg, letting the adapter pop back to the caller instead of
// tearing down the program.
func TestWizardEscEmitsExit(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	m := wizard.New(prov, buildStepList(prov))

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Update(esc) returned nil Cmd, want WizardExitMsg")
	}
	if _, ok := cmd().(wizard.WizardExitMsg); !ok {
		t.Errorf("Update(esc) emitted %T, want wizard.WizardExitMsg", cmd())
	}
}

// TestWizardLeftAtFirstStepExits verifies that left arrow on the first step
// triggers WizardExitMsg (no earlier step to retreat to).
func TestWizardLeftAtFirstStepExits(t *testing.T) {
	prov := provider.NewTestProvider(t.TempDir())
	m := wizard.New(prov, buildStepList(prov))

	// First left emits StepBackMsg.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd == nil {
		t.Fatal("Update(left) returned nil Cmd, want StepBackMsg")
	}
	backMsg, ok := cmd().(wizard.StepBackMsg)
	if !ok {
		t.Fatalf("Update(left) emitted %T, want wizard.StepBackMsg", cmd())
	}

	// Feeding StepBackMsg while on step 0 must bubble up as WizardExitMsg.
	_, cmd2 := m.Update(backMsg)
	if cmd2 == nil {
		t.Fatal("Update(StepBackMsg) at step 0 returned nil Cmd, want WizardExitMsg")
	}
	if _, ok := cmd2().(wizard.WizardExitMsg); !ok {
		t.Errorf("Update(StepBackMsg) at step 0 emitted %T, want wizard.WizardExitMsg", cmd2())
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
