// Package wizard implements the dotai setup wizard: an interactive TUI that
// guides the user through configuring Claude Code on their machine.
package wizard

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// WizardExitMsg is emitted when the user wants to leave the wizard entirely
// and return to the calling screen (typically the landing menu). The wizard
// adapter converts this message into a messages.PopScreenMsg.
type WizardExitMsg struct{}

// WizardModel is the root Bubbletea model for the dotai setup wizard.
// It owns the step list, the current phase, and orchestrates transitions
// between PhaseStepping → PhaseSummary → PhaseApplying → PhaseDone/PhaseError.
type WizardModel struct {
	steps    []Step
	current  int
	state    *WizardState
	phase    Phase
	prov     provider.Provider
	width    int
	height   int
	summary  SummaryModel
	executor *Executor
	err      error
}

// New constructs a WizardModel for the given provider and pre-built step list.
// The steps slice is typically built by the main package using the wizard/steps
// sub-package, which avoids an import cycle (wizard/steps imports wizard).
func New(prov provider.Provider, stepList []Step) WizardModel {
	state := &WizardState{Provider: prov}
	return WizardModel{
		steps:    stepList,
		current:  0,
		state:    state,
		phase:    PhaseStepping,
		prov:     prov,
		executor: NewExecutor(prov),
	}
}

// Init is called by Bubbletea when the program starts.
func (m WizardModel) Init() tea.Cmd {
	return m.steps[0].Init()
}

// Update routes messages to the active phase handler.
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global handlers — apply in every phase.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.phase == PhaseStepping && m.current < len(m.steps) {
			m.steps[m.current].SetSize(m.width, m.height)
		}
		m.summary.SetSize(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.phase {
	case PhaseStepping:
		return m.updateStepping(msg)
	case PhaseSummary:
		return m.updateSummary(msg)
	case PhaseApplying:
		return m.updateApplying(msg)
	case PhaseDone, PhaseError:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, func() tea.Msg { return WizardExitMsg{} }
		}
	}
	return m, nil
}

// updateStepping handles messages while in the stepping phase.
func (m WizardModel) updateStepping(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case StepCompleteMsg:
		m.steps[m.current].Apply(m.state)
		return m.advanceStep()
	case StepAutoSkipMsg:
		return m.advanceStep()
	case StepBackMsg:
		return m.retreatStep()
	case tea.KeyMsg:
		// Global navigation: esc exits to the caller, left goes back one step.
		// Skipped when the current step is in a sub-mode that consumes these
		// keys itself (e.g. profile preview viewport).
		if !m.stepIsCapturingKeys() {
			switch typed.String() {
			case "esc":
				return m, func() tea.Msg { return WizardExitMsg{} }
			case "left":
				return m, func() tea.Msg { return StepBackMsg{} }
			}
		}
	}

	// Forward all other messages to the active step.
	updated, cmd := m.steps[m.current].Update(msg)
	if s, ok := updated.(Step); ok {
		m.steps[m.current] = s
	}
	return m, cmd
}

// stepIsCapturingKeys reports whether the current step has signalled, via the
// optional KeyCapturing interface, that it is in a sub-mode that must receive
// esc and left directly without wizard-level interception.
func (m WizardModel) stepIsCapturingKeys() bool {
	if kc, ok := m.steps[m.current].(KeyCapturing); ok {
		return kc.IsCapturingKeys()
	}
	return false
}

// updateSummary handles messages while in the summary phase.
func (m WizardModel) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case SummaryConfirmMsg:
		if m.state.HasChanges() {
			m.phase = PhaseApplying
			return m, m.executor.Run(m.state)
		}
		m.phase = PhaseDone
		return m, nil
	case SummaryCancelMsg:
		return m, func() tea.Msg { return WizardExitMsg{} }
	case tea.KeyMsg:
		// left returns to the last step so the user can tweak their choices.
		if typed.String() == "left" {
			return m.retreatStep()
		}
	}

	updated, cmd := m.summary.Update(msg)
	if s, ok := updated.(SummaryModel); ok {
		m.summary = s
	}
	return m, cmd
}

// updateApplying handles messages while the executor is running.
func (m WizardModel) updateApplying(msg tea.Msg) (tea.Model, tea.Cmd) {
	if prog, ok := msg.(ProgressMsg); ok {
		if prog.Done {
			m.phase = PhaseDone
			return m, nil
		}
		if prog.Err != nil {
			m.phase = PhaseError
			m.err = prog.Err
			return m, nil
		}
	}
	return m, nil
}

// advanceStep moves to the next step or transitions to the summary phase.
func (m WizardModel) advanceStep() (WizardModel, tea.Cmd) {
	m.current++
	if m.current >= len(m.steps) {
		sum := NewSummary(m.state, m.steps, m.prov)
		sum.SetSize(m.width, m.height)
		m.summary = sum
		m.phase = PhaseSummary
		return m, m.summary.Init()
	}
	m.steps[m.current].SetSize(m.width, m.height)
	return m, m.steps[m.current].Init()
}

// retreatStep moves one position back in the step list, preserving whatever
// selection the previous step already holds. Behaviour by source phase:
//   - From PhaseStepping: go to the previous step; if already at the first
//     step, emit WizardExitMsg so the adapter pops back to the caller.
//   - From PhaseSummary: return to the last step so the user can edit it.
func (m WizardModel) retreatStep() (WizardModel, tea.Cmd) {
	if m.phase == PhaseSummary {
		m.phase = PhaseStepping
		m.current = len(m.steps) - 1
		m.steps[m.current].SetSize(m.width, m.height)
		return m, m.steps[m.current].Init()
	}

	if m.current == 0 {
		return m, func() tea.Msg { return WizardExitMsg{} }
	}
	m.current--
	m.steps[m.current].SetSize(m.width, m.height)
	return m, m.steps[m.current].Init()
}

// View renders the current phase.
func (m WizardModel) View() string {
	switch m.phase {
	case PhaseStepping:
		return styles.Frame(
			m.stepHeader(),
			m.steps[m.current].View(),
			m.steps[m.current].Footer(),
			m.width,
			m.height,
		)
	case PhaseSummary:
		return styles.Frame(
			m.summaryHeader(),
			m.summary.View(),
			styles.FooterHints("enter", "apply", "←", "back", "esc", "menu"),
			m.width,
			m.height,
		)
	case PhaseApplying:
		return styles.Frame("", "Applying changes...", "", m.width, m.height)
	case PhaseDone:
		return styles.Frame(
			"",
			m.doneView(),
			styles.FooterHints("any key", "exit"),
			m.width,
			m.height,
		)
	case PhaseError:
		return styles.Frame(
			"",
			m.errorView(),
			styles.FooterHints("any key", "exit"),
			m.width,
			m.height,
		)
	}
	return ""
}

// stepHeader returns the header for the current step screen.
// Format: "Step N/6 — StepName\nDescription"
func (m WizardModel) stepHeader() string {
	step := m.steps[m.current]
	counter := styles.StepProgress.Render(
		"Step " + strconv.Itoa(m.current+1) + "/" + strconv.Itoa(len(m.steps)),
	)
	name := styles.Title.Render(step.Name())
	desc := styles.Footer.Render(step.Description())
	return counter + " — " + name + "\n" + desc
}

// summaryHeader returns the header for the summary screen.
func (m WizardModel) summaryHeader() string {
	return styles.Title.Render("dotai — summary")
}

// doneView returns the body for the success screen.
func (m WizardModel) doneView() string {
	var sb strings.Builder
	sb.WriteString(styles.CheckOn.Render("✓ dotai setup complete!"))
	sb.WriteString("\n\n")
	sb.WriteString(styles.SectionTitle.Render("Installed:"))
	sb.WriteString("\n")
	if m.state.SettingsEnabled {
		sb.WriteString("  " + styles.CheckOn.Render("●") + "  Base settings\n")
	}
	if m.state.StatuslineEnabled {
		sb.WriteString("  " + styles.CheckOn.Render("●") + "  Shell status-line\n")
	}
	if m.state.GitBranchHookEnabled {
		sb.WriteString("  " + styles.CheckOn.Render("●") + "  Git branch check hook\n")
	}
	if m.state.ProfileType != ProfileNone {
		sb.WriteString("  " + styles.CheckOn.Render("●") + "  Profile (" + m.state.ProfileType.String() + ")\n")
	}
	if len(m.state.SelectedSkills) > 0 {
		sb.WriteString("  " + styles.CheckOn.Render("●") + "  Skills: " + strings.Join(m.state.SelectedSkills, ", ") + "\n")
	}
	if m.state.MemcliEnabled {
		sb.WriteString("  " + styles.CheckOn.Render("●") + "  mem-cli hooks\n")
	}
	if !m.state.HasChanges() {
		sb.WriteString(styles.Footer.Render("  (nothing was selected)"))
	}
	return sb.String()
}

// errorView returns the body for the error screen.
func (m WizardModel) errorView() string {
	return styles.ProgressError.Render("✗ Error: " + m.err.Error())
}

// Phase returns the current phase of the wizard.
func (m WizardModel) Phase() Phase {
	return m.phase
}

