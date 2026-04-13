// Package wizard — SummaryModel shows a pre-apply review screen listing all
// the changes the wizard will make. The user can confirm or cancel.
package wizard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// SummaryConfirmMsg is emitted when the user confirms they want to apply changes.
type SummaryConfirmMsg struct{}

// SummaryCancelMsg is emitted when the user cancels the wizard.
type SummaryCancelMsg struct{}

// SummaryModel renders the pre-apply review screen.
type SummaryModel struct {
	state  *WizardState
	steps  []Step
	prov   provider.Provider
	width  int
	height int
}

// NewSummary constructs a SummaryModel from the completed wizard state.
func NewSummary(state *WizardState, steps []Step, prov provider.Provider) SummaryModel {
	return SummaryModel{
		state: state,
		steps: steps,
		prov:  prov,
	}
}

// SetSize stores the current terminal dimensions.
func (m *SummaryModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init is called when the summary screen becomes active.
func (m SummaryModel) Init() tea.Cmd { return nil }

// Update handles keyboard events on the summary screen.
func (m SummaryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "y":
			return m, func() tea.Msg { return SummaryConfirmMsg{} }
		case "esc", "n", "q":
			return m, func() tea.Msg { return SummaryCancelMsg{} }
		}
	}
	return m, nil
}

// View renders the summary screen body.
func (m SummaryModel) View() string {
	var sb strings.Builder

	if !m.state.HasChanges() {
		sb.WriteString(styles.Footer.Render("Nothing to install — all steps were skipped."))
		return sb.String()
	}

	sb.WriteString(styles.SectionTitle.Render("Changes to apply:"))
	sb.WriteString("\n\n")

	for _, s := range m.steps {
		line := s.SummaryLine()
		if s.Skipped() {
			sb.WriteString(styles.SummarySkip.Render("  ○  " + s.Name() + " — " + line))
		} else {
			sb.WriteString(styles.SummaryCheck.Render("  ●  " + s.Name() + " — " + line))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
