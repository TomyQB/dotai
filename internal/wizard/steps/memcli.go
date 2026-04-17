package steps

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/wizard"
)

// memcliOption is the index of a yes/no choice for memcli installation.
type memcliOption int

const (
	memcliYes memcliOption = 0
	memcliNo  memcliOption = 1
)

// MemcliModel is the wizard step that installs memcli session-memory integration.
// Unlike other feature steps there is no provider feature guard — memcli is
// dotai's own capability and is always offered.
type MemcliModel struct {
	prov      provider.Provider
	cursor    memcliOption
	confirmed bool
	width     int
	height    int
}

// NewMemcli constructs a MemcliModel for the given provider.
func NewMemcli(prov provider.Provider) *MemcliModel {
	return &MemcliModel{prov: prov}
}

// Init has no async work for this step.
func (m *MemcliModel) Init() tea.Cmd { return nil }

// Update handles keyboard input for the memcli step.
func (m *MemcliModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = memcliYes
		case "down", "j":
			m.cursor = memcliNo
		case "tab":
			if m.cursor == memcliYes {
				m.cursor = memcliNo
			} else {
				m.cursor = memcliYes
			}
		case "enter", "right", " ":
			m.confirmed = true
			return m, func() tea.Msg { return wizard.StepCompleteMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the yes/no prompt with a description of what memcli installs.
func (m *MemcliModel) View() string {
	var b strings.Builder

	b.WriteString(styles.SectionTitle.Render("Install memcli integration?"))
	b.WriteString("\n")
	b.WriteString(styles.MenuItem.Render("Session memory skills, agents, and hooks."))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("Installs: 4 skills + doc-keeper agent + stop/pre-commit hooks"))
	b.WriteString("\n\n")

	var yesStr, noStr string
	if m.cursor == memcliYes {
		yesStr = fmt.Sprintf("%s %s", styles.Arrow.Render("▸"), styles.MenuItemSelected.Render("Yes"))
		noStr = "  " + styles.MenuItem.Render("No")
	} else {
		yesStr = "  " + styles.MenuItem.Render("Yes")
		noStr = fmt.Sprintf("%s %s", styles.Arrow.Render("▸"), styles.MenuItemSelected.Render("No"))
	}

	b.WriteString(yesStr + "\n")
	b.WriteString(noStr + "\n")

	return b.String()
}

// Apply writes memcli configuration into the wizard state when the user chose Yes.
func (m *MemcliModel) Apply(state *wizard.WizardState) {
	if !m.confirmed || m.cursor == memcliNo {
		return
	}

	state.MemcliEnabled = true

	configDir, err := m.prov.ConfigDir()
	if err != nil {
		configDir = "~/.claude"
	}
	hooksPath := configDir + "/" + m.prov.ToolDir() + "/hooks/"
	state.Patch.Hooks = append(state.Patch.Hooks,
		wizard.HookEntry{
			Event:   "Stop",
			Command: "bash " + hooksPath + wizard.MemcliStopHookScript,
		},
		wizard.HookEntry{
			Event:   "SessionStart",
			Command: "bash " + hooksPath + wizard.MemcliSessionStartHookScript,
		},
	)
}

// SummaryLine returns a human-readable one-liner for the summary screen.
func (m *MemcliModel) SummaryLine() string {
	if m.confirmed && m.cursor == memcliYes {
		return "skills + agent + hooks"
	}
	return "skipped"
}

// Skipped reports whether the step was skipped (user chose No or not yet confirmed).
func (m *MemcliModel) Skipped() bool {
	return !m.confirmed || m.cursor == memcliNo
}

// Name returns the step identifier.
func (m *MemcliModel) Name() string { return "Memcli" }

// Description returns a short explanation shown in the step header.
func (m *MemcliModel) Description() string { return "Session memory skills, agents, and hooks" }

// Footer returns key-binding hints for this step.
func (m *MemcliModel) Footer() string {
	return styles.Footer.Render("↑/↓ select • enter/→ confirm • ← back • esc menu")
}

// SetSize notifies the step of the current terminal dimensions.
func (m *MemcliModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Ensure MemcliModel satisfies the Step interface at compile time.
var _ wizard.Step = (*MemcliModel)(nil)
