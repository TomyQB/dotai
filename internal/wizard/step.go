package wizard

import tea "github.com/charmbracelet/bubbletea"

// Step extends tea.Model with the additional methods needed by WizardModel to
// compose individual interview screens into a coherent flow.
//
// Each step is responsible for:
//   - Rendering its own UI (View/Update/Init via tea.Model)
//   - Recording its answer into *WizardState when Apply() is called
//   - Reporting whether it should be skipped (Skipped())
//   - Providing a one-line summary for the summary screen (SummaryLine())
type Step interface {
	tea.Model

	// Name returns the short identifier used in progress indicators.
	Name() string

	// Description returns a one-sentence explanation shown in the step header.
	Description() string

	// Apply writes the step's answer into state. Called once, right before
	// advancing to the next step.
	Apply(state *WizardState)

	// SummaryLine returns a single line summarising the user's choice, shown
	// on the summary screen (e.g. "base settings — apply", "profile — skip").
	SummaryLine() string

	// Skipped reports whether this step was skipped (either by the user or
	// because the provider does not support the required feature).
	Skipped() bool

	// SetSize notifies the step of the current terminal dimensions so it can
	// size internal components (e.g. viewports) accordingly.
	SetSize(width, height int)

	// Footer returns optional key-binding hint text rendered below the step
	// body. Return an empty string when no hints are needed.
	Footer() string
}

// StepCompleteMsg is emitted by a step when the user has confirmed their
// choice. WizardModel advances to the next step upon receiving it.
type StepCompleteMsg struct{}

// StepAutoSkipMsg is emitted by a step during Init when the provider does not
// support the feature the step requires. WizardModel skips the step and records
// it as skipped without rendering it.
type StepAutoSkipMsg struct{}

// StepBackMsg is emitted when the user wants to return to the previous step.
// WizardModel retreats one position upon receiving it; if the current step is
// already the first one, it converts the message to a WizardExitMsg instead.
type StepBackMsg struct{}

// KeyCapturing is an optional interface a Step can implement to signal that it
// is currently in a sub-mode that consumes global navigation keys (esc, left).
// When IsCapturingKeys returns true, the wizard forwards all keys to the step
// without intercepting esc/left for its own back/exit handling. Used for modal
// states like the profile preview viewport.
type KeyCapturing interface {
	IsCapturingKeys() bool
}
